package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/internal/api/middleware"
	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/datvtph41107/bdspro/internal/service"
	"github.com/datvtph41107/bdspro/internal/token"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type organizationHTTPStoreFake struct {
	db.Store
	creates    int
	lastName   string
	accountErr error
	createErr  error
}

func (s *organizationHTTPStoreFake) GetAccountByID(_ context.Context, id int64) (db.Account, error) {
	if s.accountErr != nil {
		return db.Account{}, s.accountErr
	}
	return db.Account{ID: id, PersonID: 99}, nil
}

func (s *organizationHTTPStoreFake) CreateOrganizationTx(_ context.Context, p db.CreateOrganizationTxParams) (db.Organization, error) {
	s.creates++
	s.lastName = p.Name
	if s.createErr != nil {
		return db.Organization{}, s.createErr
	}
	return db.Organization{PartyID: 123, Name: p.Name}, nil
}

func TestStoreOrganizationHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	maker, err := token.NewJWTMaker(strings.Repeat("x", 32))
	if err != nil {
		t.Fatal(err)
	}
	access, _, err := maker.CreateToken(42, time.Minute, token.TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}

	store := &organizationHTTPStoreFake{}
	organizations := NewOrganizationHandler(service.NewOrganizationService(store))
	router := gin.New()
	router.Use(middleware.RequestIngress())
	router.POST("/organizations", middleware.Authenticate(maker), organizations.Store)

	send := func(body, bearer string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, "/organizations", strings.NewReader(body))
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
	if w := send(`{"name":"My Company"`, access); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid request") {
		t.Fatalf("malformed JSON status=%d body=%s", w.Code, w.Body.String())
	}
	if w := send(`{"name":"  "}`, access); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "name required") {
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

	store.accountErr = pgx.ErrNoRows
	if w := send(`{"name":"Example"}`, access); w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "account unavailable") {
		t.Fatalf("deleted account status=%d body=%s", w.Code, w.Body.String())
	}
	store.accountErr = nil

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
