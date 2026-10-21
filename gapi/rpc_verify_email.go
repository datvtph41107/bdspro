package gapi

import (
	"context"
	"errors"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/pb"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) VerifyEmail(ctx context.Context, req *pb.VerifyEmailRequest) (*pb.VerifyEmailResponse, error) {
	if req.GetEmailId() <= 0 || req.GetSecretCode() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid verification link")
	}
	_, err := s.store.VerifyEmailTx(ctx, db.ConsumeVerifyEmailParams{ID: req.EmailId, SecretCode: req.SecretCode})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Error(codes.InvalidArgument, "invalid or expired verification link")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "verification failed")
	}
	return &pb.VerifyEmailResponse{Verified: true}, nil
}
