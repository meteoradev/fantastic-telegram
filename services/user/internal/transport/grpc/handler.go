package grpc

import (
	"context"

	"github.com/meteoradev/fantastic-telegram/services/user/internal/port"
	pb "github.com/meteoradev/fantastic-telegram/services/user/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type userServer struct {
	pb.UnimplementedUserServiceServer
	p port.TokenProvider
}

func (s *userServer) ValidateToken(ctx context.Context, in *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	if in.Token == "" {
		return nil, status.Errorf(codes.Unauthenticated, "missing auth header")
	}
	payload, err := s.p.ValidateToken(in.Token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token")
	}
	return &pb.ValidateTokenResponse{Valid: true, UserID: payload.Claims.UserID, UserEmail: payload.Claims.Email}, nil
}

func NewUserServer(p port.TokenProvider) pb.UserServiceServer {
	return &userServer{p: p}
}

