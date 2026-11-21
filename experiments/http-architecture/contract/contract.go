// Package contract defines a small HTTP API contract shared by router spikes.
// This is NOT BDSPro production authentication or an actual team repository.
package contract

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

type CreateTeamOption struct {
	Name string `json:"name"`
}

type Team struct {
	Org  string `json:"org"`
	Name string `json:"name"`
}

type APIError struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func JSON(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, APIError{Error: message, Code: code})
}

func Health(w http.ResponseWriter, _ *http.Request) {
	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func CreateTeam(w http.ResponseWriter, r *http.Request, org string) {
	if r.Header.Get("Authorization") != "Bearer test-token" {
		Error(w, http.StatusUnauthorized, "unauthenticated", "authentication required")
		return
	}
	if org == "" {
		Error(w, http.StatusBadRequest, "invalid_request", "organization is required")
		return
	}
	// This limit applies before decoding, independent of a router's framework.
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var option CreateTeamOption
	if err := decoder.Decode(&option); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		Error(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	option.Name = strings.TrimSpace(option.Name)
	if len(option.Name) < 3 || len(option.Name) > 100 {
		Error(w, http.StatusUnprocessableEntity, "validation_failed", "name must have 3 to 100 characters")
		return
	}
	JSON(w, http.StatusCreated, Team{Org: org, Name: option.Name})
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
