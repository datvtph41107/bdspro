package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/datvtph41107/bdspro/db/mock"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestVerifyEmailAPI(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{name: "OK", status: http.StatusOK},
		{name: "Invalid", err: pgx.ErrNoRows, status: http.StatusBadRequest},
		{name: "DatabaseError", err: errors.New("db unavailable"), status: http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			store.EXPECT().VerifyEmailTx(gomock.Any(), db.ConsumeVerifyEmailParams{
				ID: 15, SecretCode: "secret",
			}).Return(db.Account{IsEmailVerified: tc.err == nil}, tc.err)
			server := newTestServer(t, store)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/users/verify_email?email_id=15&secret_code=secret", nil)
			server.router.ServeHTTP(w, req)
			require.Equal(t, tc.status, w.Code, w.Body.String())
		})
	}
}

func TestVerifyEmailBadRequest(t *testing.T) {
	server := newTestServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/users/verify_email?email_id=0", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
