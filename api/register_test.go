package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/datvtph41107/bdspro/db/mock"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateUserAPI(t *testing.T) {
	cases := []struct {
		name      string
		duplicate bool
		want      int
	}{
		{"OK", false, http.StatusCreated},
		{"DuplicateEmail", true, http.StatusConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			account := db.Account{ID: 7, PersonID: 4, Email: "first@example.test", CreatedAt: time.Now()}
			if tc.duplicate {
				store.EXPECT().RegisterTx(gomock.Any(), gomock.Any()).Return(db.Account{},
					&pgconn.PgError{Code: "23505", ConstraintName: "accounts_email_login_uq"})
			} else {
				store.EXPECT().RegisterTx(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, arg db.RegisterTxParams) (db.Account, error) {
						require.Equal(t, "first@example.test", arg.Email)
						require.NoError(t, util.CheckPassword("strong-password", arg.PasswordHash))
						return account, nil
					})
			}
			server := newTestServer(t, store)
			body, _ := json.Marshal(createUserRequest{Email: "FIRST@example.test", Password: "strong-password"})
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			server.router.ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code, w.Body.String())
			require.NotContains(t, w.Body.String(), "password_hash")
			if !tc.duplicate {
				var response userResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
				require.Equal(t, account.ID, response.AccountID)
			}
		})
	}
}
