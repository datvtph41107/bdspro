package gapi

import (
	"context"
	"errors"
	"strings"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/pb"
	"github.com/datvtph41107/bdspro/token"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) UpdateOrganization(ctx context.Context, req *pb.UpdateOrganizationRequest) (*pb.UpdateOrganizationResponse, error) {
	if req.GetOrganizationId() <= 0 || req.Name == nil || strings.TrimSpace(req.GetName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "organization id and name required")
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing access token")
	}
	headers := md.Get("authorization")
	if len(headers) != 1 {
		return nil, status.Error(codes.Unauthenticated, "missing access token")
	}
	parts := strings.Fields(headers[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization header")
	}
	payload, err := s.tokenMaker.VerifyToken(parts[1], token.TokenTypeAccess)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid access token")
	}
	organization, err := s.store.UpdateOrganization(ctx, db.UpdateOrganizationParams{
		OrganizationID: req.GetOrganizationId(),
		ActorAccountID: payload.AccountID,
		Name:           pgtype.Text{String: strings.TrimSpace(req.GetName()), Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Error(codes.PermissionDenied, "organization owner required")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "update organization failed")
	}
	return &pb.UpdateOrganizationResponse{Organization: &pb.Organization{
		PartyId:   organization.PartyID,
		Name:      organization.Name,
		CreatedAt: timestamppb.New(organization.CreatedAt),
	}}, nil
}
