package gapi

import (
	"context"
	"errors"
	"net/mail"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/datvtph41107/bdspro/pb"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	address, err := mail.ParseAddress(req.GetEmail())
	if err != nil || address.Address != req.GetEmail() || len(req.GetPassword()) < 8 || len(req.GetPassword()) > 72 {
		return nil, status.Error(codes.InvalidArgument, "invalid email or password")
	}
	account, err := auth.Register(ctx, s.store, s.verificationScheduler, req.Email, req.Password)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "accounts_email_login_uq" {
			return nil, status.Error(codes.AlreadyExists, "email already registered")
		}
		return nil, status.Error(codes.Internal, "registration failed")
	}
	return &pb.RegisterResponse{Account: convertAccount(account)}, nil
}
