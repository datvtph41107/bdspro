package gapi

import (
	"context"
	"errors"
	"strings"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/datvtph41107/bdspro/organization"
	pb "github.com/datvtph41107/bdspro/pb/bdspro/v1"
	"github.com/datvtph41107/bdspro/token"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type accountSession interface {
	AccountFromSessionHash(context.Context, string) (int64, error)
}

type workspaceLookup interface {
	ListWorkspaces(context.Context, int64) ([]organization.Workspace, error)
}

// Server is the gRPC transport adapter, not another owner of Auth or RBAC.
// Authentication is checked on every RPC against the live PostgreSQL session.
type Server struct {
	pb.UnimplementedAccountServiceServer
	sessions   accountSession
	workspaces workspaceLookup
}

type actorContextKey struct{}

func actorID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(actorContextKey{}).(int64)
	return id, ok && id > 0
}

// NewServer registers RPC services and a mandatory session-bound JWT unary
// interceptor. Returning a concrete *grpc.Server lets the caller own its
// listener and graceful shutdown; the library never binds a public port.
func NewServer(maker *token.JWTMaker, sessions accountSession, workspaces workspaceLookup) (*grpc.Server, error) {
	if maker == nil || sessions == nil || workspaces == nil {
		return nil, errors.New("gRPC requires JWT maker, session lookup and workspace lookup")
	}
	api := &Server{sessions: sessions, workspaces: workspaces}
	server := grpc.NewServer(grpc.UnaryInterceptor(func(
		ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (any, error) {
		headers, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}
		values := headers.Get("authorization")
		if len(values) != 1 {
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}
		fields := strings.Fields(values[0])
		if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}
		subject, sessionHash, err := maker.Verify(fields[1])
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}
		accountID, err := sessions.AccountFromSessionHash(ctx, sessionHash)
		if errors.Is(err, auth.ErrUnauthorized) || (err == nil && subject != accountID) {
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil, status.FromContextError(ctx.Err()).Err()
			}
			return nil, status.Error(codes.Unavailable, "authentication unavailable")
		}
		return handler(context.WithValue(ctx, actorContextKey{}, accountID), req)
	}))
	pb.RegisterAccountServiceServer(server, api)
	return server, nil
}

func (s *Server) GetCurrentAccount(ctx context.Context, _ *pb.GetCurrentAccountRequest) (*pb.GetCurrentAccountResponse, error) {
	id, ok := actorID(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	return &pb.GetCurrentAccountResponse{AccountId: id, User: &pb.User{AccountId: id}}, nil
}

func (s *Server) ListWorkspaces(ctx context.Context, _ *pb.ListWorkspacesRequest) (*pb.ListWorkspacesResponse, error) {
	id, ok := actorID(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	workspaces, err := s.workspaces.ListWorkspaces(ctx, id)
	if err != nil {
		if ctx.Err() != nil {
			return nil, status.FromContextError(ctx.Err()).Err()
		}
		return nil, status.Error(codes.Unavailable, "workspace lookup unavailable")
	}
	result := &pb.ListWorkspacesResponse{Workspaces: make([]*pb.Workspace, 0, len(workspaces))}
	for _, workspace := range workspaces {
		result.Workspaces = append(result.Workspaces, &pb.Workspace{
			OrganizationId: workspace.OrganizationID,
			OrganizationName: workspace.OrganizationName,
			MembershipId: workspace.MembershipID,
			JoinedAt: timestamppb.New(workspace.JoinedAt),
		})
	}
	return result, nil
}
