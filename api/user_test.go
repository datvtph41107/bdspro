package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/datvtph41107/bdspro/db/mock"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5"
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

func TestLoginUserAPI(t *testing.T) {
	hash, err := util.HashPassword("correct-password")
	require.NoError(t, err)
	user := db.Account{ID: 17, Email: "test@example.test", PasswordHash: hash, CreatedAt: time.Now()}
	cases := []struct {
		name, password string
		dbErr          error
		want           int
	}{
		{"OK", "correct-password", nil, http.StatusOK},
		{"IncorrectPassword", "wrong-password", nil, http.StatusUnauthorized},
		{"UserNotFound", "correct-password", pgx.ErrNoRows, http.StatusUnauthorized},
		{"DBError", "correct-password", errors.New("database down"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			if tc.dbErr != nil {
				store.EXPECT().GetAccountByEmail(gomock.Any(), user.Email).Return(db.Account{}, tc.dbErr)
			} else {
				store.EXPECT().GetAccountByEmail(gomock.Any(), user.Email).Return(user, nil)
				if tc.want == http.StatusOK {
					store.EXPECT().CreateSession(gomock.Any(), gomock.Any()).DoAndReturn(
						func(_ context.Context, arg db.CreateSessionParams) (db.Session, error) {
							require.Equal(t, user.ID, arg.AccountID)
							require.NotEmpty(t, arg.RefreshToken)
							require.True(t, arg.ExpiresAt.After(time.Now()))
							return db.Session{ID: arg.ID, AccountID: arg.AccountID, RefreshToken: arg.RefreshToken, ExpiresAt: arg.ExpiresAt}, nil
						},
					)
				}
			}
			server := newTestServer(t, store)
			body, _ := json.Marshal(loginUserRequest{Email: user.Email, Password: tc.password})
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/users/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			server.router.ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code, w.Body.String())
			if tc.want == http.StatusOK {
				var rsp loginUserResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rsp))
				require.NotEmpty(t, rsp.AccessToken)
				require.NotEmpty(t, rsp.RefreshToken)
				require.NotEmpty(t, rsp.SessionID)
				require.True(t, rsp.RefreshTokenExpiresAt.After(rsp.AccessTokenExpiresAt))
			}
		})
	}
}

func TestCreateUserBadRequest(t *testing.T) {
	server := newTestServer(t, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginSessionInsertFailure(t *testing.T) {
	hash, err := util.HashPassword("strong-password")
	require.NoError(t, err)
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetAccountByEmail(gomock.Any(), "abc@example.test").Return(
		db.Account{ID: 5, Email: "abc@example.test", PasswordHash: hash}, nil,
	)
	store.EXPECT().CreateSession(gomock.Any(), gomock.Any()).Return(
		db.Session{}, errors.New("postgres-secret-do-not-leak"),
	)
	server := newTestServer(t, store)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/login", bytes.NewBufferString(
		`{"email":"abc@example.test","password":"strong-password"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.NotContains(t, w.Body.String(), "postgres-secret")
}
