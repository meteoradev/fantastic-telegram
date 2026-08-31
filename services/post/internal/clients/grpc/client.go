package client

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/meteoradev/fantastic-telegram/services/post/internal/domain"
	pb "github.com/meteoradev/fantastic-telegram/services/post/proto"
)

type grpcClient struct {
	grpcClient pb.UserServiceClient
	Conn       *grpc.ClientConn
	timeout    time.Duration
}

func NewGRPCClient(serverAddr string, opts ...grpc.DialOption) (*grpcClient, error) {
	conn, err := grpc.NewClient(serverAddr,
		append([]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, opts...)...,
	)
	if err != nil {
		return nil, err
	}
	return &grpcClient{
		grpcClient: pb.NewUserServiceClient(conn),
		Conn:       conn,
		timeout:    5 * time.Second,
	}, nil
}

func (c *grpcClient) Close() error {
	return c.Conn.Close()
}

func (c *grpcClient) ValidateToken(ctx context.Context, token string) (*domain.Claims, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := &pb.ValidateTokenRequest{
		Token: token,
	}

	resp, err := c.grpcClient.ValidateToken(ctx, req)
	if err != nil {
		return nil, err
	}
	if !resp.Valid {
		return nil, domain.ErrInvalidToken
	}
	return &domain.Claims{
		UserID: resp.UserID,
		Email:  resp.UserEmail,
	}, nil
}

