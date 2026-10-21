package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/datvtph41107/bdspro/db/mock"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func authRequest(t *testing.T, server *Server, method, path, body string) *http.Request {
	t.Helper()
	jwt, _, err := server.tokenMaker.CreateToken(10, time.Hour, token.TokenTypeAccess)
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestCreateOrganizationAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetAccountByID(gomock.Any(), int64(10)).Return(db.Account{ID: 10, PersonID: 4}, nil)
	store.EXPECT().CreateOrganizationTx(gomock.Any(), db.CreateOrganizationTxParams{
		Name: "Example Company", OwnerPersonID: 4,
	}).Return(db.Organization{PartyID: 20, Name: "Example Company"}, nil)
	server := newTestServer(t, store)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, authRequest(t, server, http.MethodPost, "/organizations", `{"name":" Example Company "}`))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
}

func TestAddOrganizationMemberRequiresOwner(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().AddOrganizationMember(gomock.Any(), db.AddOrganizationMemberParams{
		OrganizationID: 20, PersonID: 17, ActorAccountID: 10,
	}).DoAndReturn(func(_ context.Context, _ db.AddOrganizationMemberParams) (db.OrganizationMembership, error) {
		return db.OrganizationMembership{}, pgx.ErrNoRows
	})
	server := newTestServer(t, store)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, authRequest(t, server, http.MethodPost, "/organizations/20/members", `{"person_id":17}`))
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
}

func TestUpdateOrganizationAPI(t *testing.T) {
	tests := []struct {
		name string
		dbErr error
		want int
	}{
		{name: "Owner", want: http.StatusOK},
		{name: "NotOwner", dbErr: pgx.ErrNoRows, want: http.StatusForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			store.EXPECT().UpdateOrganization(gomock.Any(), db.UpdateOrganizationParams{
				OrganizationID: 20,
				ActorAccountID: 10,
				Name: pgtype.Text{String: "Updated Organization", Valid: true},
			}).Return(db.Organization{PartyID: 20, Name: "Updated Organization"}, tc.dbErr)
			server := newTestServer(t, store)
			w := httptest.NewRecorder()
			req := authRequest(t, server, http.MethodPatch, "/organizations/20", `{"name":" Updated Organization "}`)
			server.router.ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}
}

func TestUpdateOrganizationBadRequest(t *testing.T) {
	server := newTestServer(t, nil)
	for _, body := range []string{`{}`, `{"name":""}`, `{"name":"   "}`} {
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, authRequest(t, server, http.MethodPatch, "/organizations/20", body))
		require.Equal(t, http.StatusBadRequest, w.Code)
	}
}
