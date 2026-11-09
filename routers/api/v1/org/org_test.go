package org

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/modules/web/middleware"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type organizationHTTPStoreFake struct {
	db.Store
	creates    int
	lastName   string
	accountErr error
	sessionErr error
	createErr  error
	sessionID  uuid.UUID
	lockOrder  []string
}

// The fake runs the actual SQLC methods through a small recording DBTX.
// It does not reimplement the service's Party -> Organization -> owner workflow.
func (s *organizationHTTPStoreFake) ExecTx(ctx context.Context, operation func(*db.Queries) error) error {
	s.creates++
	if s.createErr != nil {
		return s.createErr
	}
	return operation(db.New(organizationHTTPQueries{store: s}))
}

type organizationHTTPQueries struct {
	store *organizationHTTPStoreFake
}

func (q organizationHTTPQueries) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	return organizationHTTPRow{sql: sql, args: args, store: q.store}
}

func (organizationHTTPQueries) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("unexpected Exec in creation workflow")
}

func (organizationHTTPQueries) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected Query in creation workflow")
}

type organizationHTTPRow struct {
	sql   string
	args  []any
	store *organizationHTTPStoreFake
}

func (r organizationHTTPRow) Scan(dest ...any) error {
	switch {
	case strings.Contains(r.sql, "-- name: LockOrganizationCreator"):
		if len(r.args) != 1 || r.args[0] != int64(42) {
			return errors.New("unexpected creator account ID")
		}
		if r.store.accountErr != nil {
			return r.store.accountErr
		}
		r.store.lockOrder = append(r.store.lockOrder, "account")
		*dest[0].(*int64) = 99
	case strings.Contains(r.sql, "-- name: LockActingSession"):
		if len(r.args) != 2 || r.args[0] != r.store.sessionID || r.args[1] != int64(42) {
			return errors.New("unexpected acting session/account")
		}
		if len(r.store.lockOrder) == 0 || r.store.lockOrder[len(r.store.lockOrder)-1] != "account" {
			return errors.New("acting session locked before creator account")
		}
		if r.store.sessionErr != nil {
			return r.store.sessionErr
		}
		r.store.lockOrder = append(r.store.lockOrder, "session")
		*dest[0].(*uuid.UUID) = r.store.sessionID
	case strings.Contains(r.sql, "INSERT INTO parties DEFAULT VALUES"):
		*dest[0].(*int64) = 123
	case strings.Contains(r.sql, "INSERT INTO organizations (party_id, name)"):
		r.store.lastName = r.args[1].(string)
		*dest[0].(*int64) = 123
		*dest[1].(*string) = r.store.lastName
		*dest[2].(*time.Time) = time.Now()
	case strings.Contains(r.sql, "INSERT INTO organization_members (organization_id, person_id, role, owner_person_id)"):
		if r.args[0] != int64(123) || r.args[1] != int64(99) || r.args[2] != "owner" {
			return errors.New("unexpected owner membership arguments")
		}
		*dest[0].(*int64) = 1
		*dest[1].(*int64) = 123
		*dest[2].(*int64) = 99
		*dest[3].(*string) = "owner"
		*dest[4].(*time.Time) = time.Now()
		*dest[6].(*pgtype.Int8) = pgtype.Int8{Int64: 99, Valid: true}
	default:
		return errors.New("unexpected SQLC statement")
	}
	return nil
}

func TestStoreOrganizationHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("x", 32))
	if err != nil {
		t.Fatal(err)
	}
	sessionID := uuid.New()
	access, _, err := maker.CreateSessionAccessToken(42, time.Minute, sessionID)
	if err != nil {
		t.Fatal(err)
	}

	store := &organizationHTTPStoreFake{sessionID: sessionID}
	router := gin.New()
	router.Use(appctx.RequestIngress(), appctx.APIContexter())
	router.POST("/orgs", appctx.Authenticate(maker), func(c *gin.Context) {
		payload, ok := appctx.AuthorizationPayload(c)
		if !ok {
			appctx.AuthenticationRequired(c)
			return
		}
		scope := db.WithStore(c.Request.Context(), store)
		scope = auth_model.WithActor(scope, auth_model.Actor{
			AccountID: payload.AccountID, SessionID: payload.SessionID,
		})
		c.Request = c.Request.WithContext(scope)
		api := appctx.GetAPIContext(c)
		api.IsSigned = true
		api.Doer = payload
		c.Next()
	}, func(c *gin.Context) {
		api := appctx.GetAPIContext(c)
		form, err := middleware.BindJSON[structs.CreateOrgOption](api)
		if err != nil {
			api.Validation(err, form)
			return
		}
		web.SetForm(api, form)
		c.Next()
	}, appctx.Wrap(Create))

	send := func(body, bearer string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, "/orgs", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		if bearer != "" {
			request.Header.Set("Authorization", "Bearer "+bearer)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, request)
		return w
	}

	if w := send(`{"name":"My Company"}`, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status=%d body=%s", w.Code, w.Body.String())
	}
	if store.creates != 0 {
		t.Fatal("unauthenticated request reached database")
	}
	if w := send(`{"name":"My Company"`, access); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `"code":"malformed_json"`) {
		t.Fatalf("malformed JSON status=%d body=%s", w.Code, w.Body.String())
	}
	if w := send(`{"name":"  "}`, access); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `"code":"validation_failed"`) || !strings.Contains(w.Body.String(), `"name":[{"code":"required"`) {
		t.Fatalf("blank name status=%d body=%s", w.Code, w.Body.String())
	}
	if store.creates != 0 {
		t.Fatal("invalid input reached database")
	}
	if w := send(`{"name":"  Example  "}`, access); w.Code != http.StatusCreated || !strings.Contains(w.Body.String(), `"name":"Example"`) {
		t.Fatalf("success status=%d body=%s", w.Code, w.Body.String())
	}
	if store.creates != 1 || store.lastName != "Example" {
		t.Fatalf("unexpected DB creation: calls=%d name=%q", store.creates, store.lastName)
	}
	if len(store.lockOrder) != 2 || store.lockOrder[0] != "account" || store.lockOrder[1] != "session" {
		t.Fatalf("wrong account/session lock order: %v", store.lockOrder)
	}

	store.accountErr = pgx.ErrNoRows
	if w := send(`{"name":"Example"}`, access); w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "account unavailable") {
		t.Fatalf("deleted account status=%d body=%s", w.Code, w.Body.String())
	}
	store.accountErr = nil

	store.sessionErr = pgx.ErrNoRows
	if w := send(`{"name":"Example"}`, access); w.Code != http.StatusUnauthorized {
		t.Fatalf("revoked acting session status=%d body=%s", w.Code, w.Body.String())
	}
	store.sessionErr = nil

	store.createErr = errors.New("postgres credential must not be sent")
	if w := send(`{"name":"Example"}`, access); w.Code != http.StatusInternalServerError ||
		!strings.Contains(w.Body.String(), "internal server error") ||
		strings.Contains(w.Body.String(), "postgres credential") {
		t.Fatalf("unexpected internal error response status=%d body=%s", w.Code, w.Body.String())
	}
	if w := send(strings.Repeat("a", (1<<20)+1), access); w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body status=%d body=%s", w.Code, w.Body.String())
	}
}
