// @title           Blog API
// @version         1.0
// @description     REST API РґР»СЏ Р±Р»РѕРіР° СЃ Р°РІС‚РѕСЂРёР·Р°С†РёРµР№, СѓРїСЂР°РІР»РµРЅРёРµРј РїРѕР»СЊР·РѕРІР°С‚РµР»СЏРјРё Рё РїРѕСЃС‚Р°РјРё
// @host            localhost:8080
// @BasePath        /
// @schemes         http
// @securityDefinitions.apikey BearerAuth
// @in                        header
// @name                      Authorization
// @description               Р’РІРµРґРёС‚Рµ С‚РѕРєРµРЅ РІ С„РѕСЂРјР°С‚Рµ: Bearer <РІР°С€_С‚РѕРєРµРЅ>

package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/meteoradev/fantastic-telegram/services/user/config"
	_ "github.com/meteoradev/fantastic-telegram/services/user/docs"
	"github.com/meteoradev/fantastic-telegram/services/user/internal/infra/hasher"
	"github.com/meteoradev/fantastic-telegram/services/user/internal/infra/jwt"
	"github.com/meteoradev/fantastic-telegram/services/user/internal/repository/postgres"
	"github.com/meteoradev/fantastic-telegram/services/user/internal/repository/redis"
	gr "github.com/meteoradev/fantastic-telegram/services/user/internal/transport/grpc"
	"google.golang.org/grpc"

	"github.com/fatih/color"
	"github.com/meteoradev/fantastic-telegram/services/user/internal/service"
	"github.com/meteoradev/fantastic-telegram/services/user/internal/transport/http/handler"
	pb "github.com/meteoradev/fantastic-telegram/services/user/proto"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	httpSwagger "github.com/swaggo/http-swagger"
)

func printBanner(cfg *config.Config) {
	color.NoColor = false // Force colors

	banner := `
 ________ ________  ________   _________  ________  ________  _________  ___  ________     
|\  _____\\   __  \|\   ___  \|\___   ___\\   __  \|\   ____\|\___   ___\\  \|\   ____\    
\ \  \__/\ \  \|\  \ \  \\ \  \|___ \  \_\ \  \|\  \ \  \___|\|___ \  \_\ \  \ \  \___|    
 \ \   __\\ \   __  \ \  \\ \  \   \ \  \ \ \   __  \ \_____  \   \ \  \ \ \  \ \  \       
  \ \  \_| \ \  \ \  \ \  \\ \  \   \ \  \ \ \  \ \  \|____|\  \   \ \  \ \ \  \ \  \____  
   \ \__\   \ \__\ \__\ \__\\ \__\   \ \__\ \ \__\ \__\____\_\  \   \ \__\ \ \__\ \_______\
    \|__|    \|__|\|__|\|__| \|__|    \|__|  \|__|\|__|\_________\   \|__|  \|__|\|_______|
                                                      \|_________|                         
 _________  _______   ___       _______   ________  ________  ________  _____ ______       
|\___   ___\\  ___ \ |\  \     |\  ___ \ |\   ____\|\   __  \|\   __  \|\   _ \  _   \     
\|___ \  \_\ \   __/|\ \  \    \ \   __/|\ \  \___|\ \  \|\  \ \  \|\  \ \  \\\__\ \  \    
     \ \  \ \ \  \_|/_\ \  \    \ \  \_|/_\ \  \  __\ \   _  _\ \   __  \ \  \\|__| \  \   
      \ \  \ \ \  \_|\ \ \  \____\ \  \_|\ \ \  \|\  \ \  \\  \\ \  \ \  \ \  \    \ \  \  
       \ \__\ \ \_______\ \_______\ \_______\ \_______\ \__\\ _\\ \__\ \__\ \__\    \ \__\ 
        \|__|  \|_______|\|_______|\|_______|\|_______|\|__|\|__|\|__|\|__|\|__|     \|__|
	`
	red := color.New(color.FgRed, color.Bold)
	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	magenta := color.New(color.FgMagenta, color.Bold)
	white := color.New(color.FgWhite, color.Bold)

	red.Println(banner)
	println()

	http_port := cfg.HttpPort

	yellow.Println("Configuration:")
	white.Printf("   Port:        %d\n", http_port)
	white.Printf("   JWT Expiry:  %s\n", time.Duration(cfg.Expiry).String())
	white.Printf("   Rate Limit:  Public=%d RPM, Protected=%d RPM\n", cfg.PublicRPM, cfg.ProtectedRPM)
	println()
	green.Println("Services:")
	green.Println("   вњ“ PostgreSQL connected")
	green.Println("   вњ“ Redis connected")
	green.Println("   вњ“ JWT Auth middleware enabled")
	green.Println("   вњ“ Rate limiting enabled")
	green.Println("   вњ“ Logging middleware enabled")
	green.Println("   вњ“ Recovery middleware enabled")
	println()
	magenta.Println("User URLs:")
	white.Printf("   API Base:     http://localhost:%d/\n", http_port)
	white.Printf("   Swagger:      http://localhost:%d/swagger/\n", http_port)
	white.Printf("   Swagger JSON: http://localhost:%d/swagger/doc.json\n", http_port)
	println()
	magenta.Println("Post URLs:")
	white.Printf("   API Base:     http://localhost:%d/\n", 8081)
	white.Printf("   Swagger:      http://localhost:%d/swagger/\n", 8081)
	white.Printf("   Swagger JSON: http://localhost:%d/swagger/doc.json\n", 8081)
}

func main() {

	// Logging
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger().Output(zerolog.ConsoleWriter{Out: os.Stdout})
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	logger.Info().Msg("Starting the application.")

	// Load cfg
	if _, err := os.Stat("../../.env"); err == nil {
		err := godotenv.Load("../../.env")
		if err != nil {
			logger.Fatal().Err(err).Msg("Fatal error during parse env file.")
		}
	}
	cfg := config.Load()

	// connect to cache
	rdb, err := redis.RedisConnect(cfg)
	if err != nil {
		logger.Err(err).Str("component", "Redis").Msg("Redis could not connect to db.")
	}

	// connect to DB
	DB, err := postgres.NewPostgresDB(cfg)
	if err != nil {
		logger.Fatal().Err(err).Str("component", "Postgres").Msg("Postgres could not connect to db.")
	}

	// prepare user controller
	hasher := hasher.NewBcryptHasher(0)
	userRepo := postgres.NewUserRepository(DB)
	userSVC := service.NewUserService(userRepo, hasher)
	userCtrl := handler.NewUserController(userSVC)

	// prepare auth controller
	prov := jwt.NewProvider(cfg.SecretKey, time.Duration(cfg.Expiry))
	authSVC := service.NewAuthService(userRepo, hasher, prov)
	authCtrl := handler.NewAuthController(authSVC)

	r := handler.NewRouter(rdb, userCtrl, authCtrl, cfg.SecretKey, time.Duration(cfg.Expiry), cfg.PublicRPM, cfg.ProtectedRPM, logger)

	// Swagger
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	httpSrv := &http.Server{
		Addr:    ":" + strconv.FormatInt(int64(cfg.HttpPort), 10),
		Handler: r,
	}
	grpcSrv := grpc.NewServer()

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().
				Err(err).
				Msg("Critical error during starting server")
		}
	}()
	go func() {
		l, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(cfg.GrpcPort), 10))
		if err != nil {
			logger.Fatal().
				Err(err).
				Msg("Critical error during starting server")
		}
		pb.RegisterUserServiceServer(grpcSrv, gr.NewUserServer(jwt.NewProvider(cfg.SecretKey, time.Duration(cfg.Expiry))))
		grpcSrv.Serve(l)
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	printBanner(cfg)
	<-ctx.Done()

	// Shutdown
	logger.Info().Msg("Shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal().
			Err(err).
			Msg("Server shutdown failed")
	}
	grpcSrv.GracefulStop()
	DB.Close()
	rdb.Close()

}
