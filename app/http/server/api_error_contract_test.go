package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mockdb "github.com/datvtph41107/bdspro/db/mock"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestPositiveInt64ParamContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/resource/:id", func(ctx *gin.Context) {
		id, ok := positiveInt64Param(ctx, "id")
		if !ok {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid resource"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"id": id})
	})
	cases := []struct {
		id, body string
		status   int
	}{
		{"42", `{"id":42}`, http.StatusOK},
		{"00042", `{"id":42}`, http.StatusOK},
		{"0", `{"error":"invalid resource"}`, http.StatusBadRequest},
		{"-1", `{"error":"invalid resource"}`, http.StatusBadRequest},
		{"abc", `{"error":"invalid resource"}`, http.StatusBadRequest},
		{"9223372036854775808", `{"error":"invalid resource"}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/resource/"+tc.id, nil))
			require.Equal(t, tc.status, w.Code, w.Body.String())
			require.JSONEq(t, tc.body, w.Body.String())
		})
	}
}

func TestPostgresErrorUnwrapContract(t *testing.T) {
	require.Nil(t, postgresError(nil))
	require.Nil(t, postgresError(errors.New("redis unavailable")))
	pgErr := &pgconn.PgError{Code: "23505", ConstraintName: "accounts_email_login_uq"}
	require.Same(t, pgErr, postgresError(pgErr))
	require.Same(t, pgErr, postgresError(fmt.Errorf("transaction: %w", pgErr)))
}

func TestOrganizationRouteIDsRejectInvalidValuesBeforeStore(t *testing.T) {
	app := newTestServer(t, nil)
	for _, id := range []string{"0", "-1", "not-a-number", "9223372036854775808"} {
		t.Run(id, func(t *testing.T) {
			for _, tc := range []struct {
				method, path, body string
			}{
				{http.MethodPatch, "/organizations/" + id, `{"name":"Valid Name"}`},
				{http.MethodPost, "/organizations/" + id + "/members", `{"person_id":17}`},
			} {
				w := httptest.NewRecorder()
				app.Handler().ServeHTTP(w, authRequest(t, app, tc.method, tc.path, tc.body))
				require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
				require.JSONEq(t, `{"error":"invalid organization"}`, w.Body.String())
			}
		})
	}
}

func TestAddOrganizationMemberPostgresErrorResponses(t *testing.T) {
	for _, tc := range []struct {
		name     string
		err      error
		status   int
		response string
	}{
		{
			name: "ActiveMembershipConflict", err: &pgconn.PgError{Code: "23505"},
			status: http.StatusConflict, response: "membership already exists",
		},
		{
			name: "UnavailablePerson", err: &pgconn.PgError{Code: "23503"},
			status: http.StatusNotFound, response: "person unavailable",
		},
		{
			name: "OtherConstraint", err: &pgconn.PgError{Code: "23514"},
			status: http.StatusInternalServerError, response: "add member failed",
		},
		{
			name: "WrappedUniqueness", err: fmt.Errorf("insert membership: %w", &pgconn.PgError{Code: "23505"}),
			status: http.StatusConflict, response: "membership already exists",
		},
		{
			name: "InfrastructureFailure", err: errors.New("internal-postgres-secret"),
			status: http.StatusInternalServerError, response: "add member failed",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			store.EXPECT().AddOrganizationMember(gomock.Any(), db.AddOrganizationMemberParams{
				OrganizationID: 20, PersonID: 17, ActorAccountID: 10,
			}).Return(db.OrganizationMembership{}, tc.err)
			app := newTestServer(t, store)
			w := httptest.NewRecorder()
			app.Handler().ServeHTTP(w, authRequest(t, app, http.MethodPost,
				"/organizations/20/members", `{"person_id":17}`))
			require.Equal(t, tc.status, w.Code, w.Body.String())
			require.JSONEq(t, fmt.Sprintf(`{"error":%q}`, tc.response), w.Body.String())
			require.NotContains(t, strings.ToLower(w.Body.String()), "internal-postgres-secret")
		})
	}
}

func TestRegisterUniqueConstraintIsScopedToEmail(t *testing.T) {
	for _, tc := range []struct {
		name, constraint string
		status           int
		message          string
	}{
		{"EmailConflict", "accounts_email_login_uq", http.StatusConflict, "email already registered"},
		{"UnrelatedConflict", "some_other_unique_constraint", http.StatusInternalServerError, "registration failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			store.EXPECT().RegisterTx(gomock.Any(), gomock.Any()).Return(
				db.Account{}, fmt.Errorf("register: %w", &pgconn.PgError{Code: "23505", ConstraintName: tc.constraint}),
			)
			app := newTestServer(t, store)
			req := httptest.NewRequest(http.MethodPost, "/users",
				strings.NewReader(`{"email":"sample@example.test","password":"strong-password"}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			app.Handler().ServeHTTP(w, req)
			require.Equal(t, tc.status, w.Code, w.Body.String())
			require.JSONEq(t, fmt.Sprintf(`{"error":%q}`, tc.message), w.Body.String())
		})
	}
}
