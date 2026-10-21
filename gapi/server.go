package gapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/mail"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/auth"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/pb"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
	"github.com/datvtph41107/bdspro/worker"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedBDSProServer
	config          util.Config
	store           db.Store
	tokenMaker      token.Maker
	taskDistributor worker.TaskDistributor
}

func NewServer(config util.Config, store db.Store, distributor worker.TaskDistributor) (*Server, error) {
	maker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("create token maker: %w", err)
	}
	return &Server{config: config, store: store, tokenMaker: maker, taskDistributor: distributor}, nil
}

func convertAccount(account db.Account) *pb.Account {
	return &pb.Account{
		AccountId: account.ID, Email: account.Email,
		CreatedAt: timestamppb.New(account.CreatedAt), IsEmailVerified: account.IsEmailVerified,
	}
}

func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	address, err := mail.ParseAddress(req.GetEmail())
	if err != nil || address.Address != req.GetEmail() || len(req.GetPassword()) < 8 || len(req.GetPassword()) > 72 {
		return nil, status.Error(codes.InvalidArgument, "invalid email or password")
	}
	account, err := auth.Register(ctx, s.store, s.taskDistributor, req.Email, req.Password)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "accounts_email_login_uq" {
			return nil, status.Error(codes.AlreadyExists, "email already registered")
		}
		return nil, status.Error(codes.Internal, "registration failed")
	}
	return &pb.RegisterResponse{Account: convertAccount(account)}, nil
}

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

func unaryLogger(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	response, err := handler(ctx, req)
	slog.InfoContext(ctx, "gRPC request", "method", info.FullMethod,
		"code", status.Code(err), "duration", time.Since(start))
	return response, err
}

func (s *Server) Start(ctx context.Context, address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(unaryLogger))
	pb.RegisterBDSProServer(grpcServer, s)
	reflection.Register(grpcServer)
	errorsCh := make(chan error, 1)
	go func() { errorsCh <- grpcServer.Serve(listener) }()

	select {
	case err := <-errorsCh:
		if errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return err
	case <-ctx.Done():
		grpcServer.GracefulStop()
		err := <-errorsCh
		if errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return err
	}
}
