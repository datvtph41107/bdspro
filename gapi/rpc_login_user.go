package gapi

import (
	"context"
	"errors"
	"strings"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/datvtph41107/bdspro/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password required")
	}
	var userAgent, clientIP string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get("user-agent"); len(values) != 0 {
			userAgent = values[0]
		}
		if values := md.Get("x-forwarded-for"); len(values) != 0 {
			clientIP = strings.TrimSpace(strings.Split(values[0], ",")[0])
		}
	}
	result, err := auth.Login(ctx, s.store, s.tokenMaker, s.config, req.Email, req.Password, userAgent, clientIP)
	if errors.Is(err, auth.ErrInvalidCredentials) {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "login failed")
	}
	return &pb.LoginResponse{
		SessionId:   result.Session.ID.String(),
		AccessToken: result.AccessToken, AccessTokenExpiresAt: timestamppb.New(result.AccessExpiresAt),
		RefreshToken: result.RefreshToken, RefreshTokenExpiresAt: timestamppb.New(result.RefreshExpiresAt),
		Account: convertAccount(result.Account),
	}, nil
}
