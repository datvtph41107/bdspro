package gapi

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	mockdb "github.com/datvtph41107/bdspro/db/mock"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/pb"
	"github.com/datvtph41107/bdspro/util"
	"github.com/datvtph41107/bdspro/worker"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/metadata"
	"github.com/jackc/pgx/v5/pgtype"
)

type testTaskDistributor struct{}

func (testTaskDistributor) DistributeTaskSendVerifyEmail(_ context.Context, _ *worker.PayloadSendVerifyEmail, _ ...asynq.Option) error {
	return nil
}

func newTestServer(t *testing.T, store db.Store) *Server {
	t.Helper()
	server, err := NewServer(util.Config{
		TokenSymmetricKey:    strings.Repeat("s", 32),
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
	}, store, testTaskDistributor{})
	require.NoError(t, err)
	return server
}

func TestRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().RegisterTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, arg db.RegisterTxParams) (db.Account, error) {
			require.Equal(t, "person@example.test", arg.Email)
			require.NoError(t, util.CheckPassword("password-123", arg.PasswordHash))
			require.NotNil(t, arg.AfterCreate)
			return db.Account{ID: 12, Email: arg.Email, CreatedAt: time.Now()}, nil
		},
	)
	reply, err := newTestServer(t, store).Register(context.Background(), &pb.RegisterRequest{
		Email: "person@example.test", Password: "password-123",
	})
	require.NoError(t, err)
	require.Equal(t, int64(12), reply.Account.AccountId)
}

func TestRegisterRejectsInvalidInput(t *testing.T) {
	_, err := newTestServer(t, nil).Register(context.Background(), &pb.RegisterRequest{
		Email: "not-email", Password: "short",
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestLogin(t *testing.T) {
	hash, err := util.HashPassword("password-123")
	require.NoError(t, err)
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetAccountByEmail(gomock.Any(), "person@example.test").Return(
		db.Account{ID: 12, Email: "person@example.test", PasswordHash: hash}, nil,
	)
	store.EXPECT().CreateSession(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, params db.CreateSessionParams) (db.Session, error) {
			require.Equal(t, int64(12), params.AccountID)
			return db.Session{ID: params.ID}, nil
		},
	)
	reply, err := newTestServer(t, store).Login(context.Background(), &pb.LoginRequest{
		Email: "person@example.test", Password: "password-123",
	})
	require.NoError(t, err)
	require.NotEmpty(t, reply.AccessToken)
	require.NotEmpty(t, reply.RefreshToken)
	require.NotNil(t, reply.Account)
}

func TestLoginMissingAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetAccountByEmail(gomock.Any(), "missing@example.test").Return(db.Account{}, pgx.ErrNoRows)
	_, err := newTestServer(t, store).Login(context.Background(), &pb.LoginRequest{
		Email: "missing@example.test", Password: "password-123",
	})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestVerifyEmail(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		code codes.Code
	}{
		{name: "OK", code: codes.OK},
		{name: "Unknown", err: pgx.ErrNoRows, code: codes.InvalidArgument},
		{name: "DBFailure", err: errors.New("db unavailable"), code: codes.Internal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			store.EXPECT().VerifyEmailTx(gomock.Any(), db.ConsumeVerifyEmailParams{
				ID: 5, SecretCode: "code",
			}).Return(db.Account{}, tc.err)
			reply, err := newTestServer(t, store).VerifyEmail(context.Background(),
				&pb.VerifyEmailRequest{EmailId: 5, SecretCode: "code"})
			require.Equal(t, tc.code, status.Code(err))
			if tc.err == nil {
				require.True(t, reply.Verified)
			}
		})
	}
}

func TestUpdateOrganizationRPC(t *testing.T) {
	for _, tc := range []struct {
		name string
		dbErr error
		want codes.Code
	}{
		{name: "Owner", want: codes.OK},
		{name: "NotOwner", dbErr: pgx.ErrNoRows, want: codes.PermissionDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			store.EXPECT().UpdateOrganization(gomock.Any(), db.UpdateOrganizationParams{
				OrganizationID: 21, ActorAccountID: 77,
				Name: pgtype.Text{String: "New Name", Valid: true},
			}).Return(db.Organization{PartyID: 21, Name: "New Name", CreatedAt: time.Now()}, tc.dbErr)
			server := newTestServer(t, store)
			raw, _, err := server.tokenMaker.CreateToken(77, time.Hour, token.TokenTypeAccess)
			require.NoError(t, err)
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+raw))
			name := " New Name "
			response, err := server.UpdateOrganization(ctx, &pb.UpdateOrganizationRequest{OrganizationId: 21, Name: &name})
			require.Equal(t, tc.want, status.Code(err))
			if tc.dbErr == nil {
				require.Equal(t, int64(21), response.Organization.PartyId)
				require.Equal(t, "New Name", response.Organization.Name)
			}
		})
	}
}

func TestUpdateOrganizationRPCRequiresAccessToken(t *testing.T) {
	server := newTestServer(t, nil)
	name := "New Name"
	_, err := server.UpdateOrganization(context.Background(), &pb.UpdateOrganizationRequest{
		OrganizationId: 21, Name: &name,
	})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	raw, _, err := server.tokenMaker.CreateToken(77, time.Hour, token.TokenTypeRefresh)
	require.NoError(t, err)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+raw))
	_, err = server.UpdateOrganization(ctx, &pb.UpdateOrganizationRequest{OrganizationId: 21, Name: &name})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestUpdateOrganizationRPCRequiresName(t *testing.T) {
	server := newTestServer(t, nil)
	_, err := server.UpdateOrganization(context.Background(), &pb.UpdateOrganizationRequest{OrganizationId: 21})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}
