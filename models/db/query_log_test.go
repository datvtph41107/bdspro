package db

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
)

func TestQueryLoggingPolicyAndPrivacy(t *testing.T) {
	for _, tc := range []struct {
		name, mode string
		slow       bool
		err        error
		level      string
	}{
		{"off", "off", true, &pgconn.PgError{Code: "23505", Message: "secret-error", Detail: "secret-detail"}, ""},
		{"fast", "slow", false, nil, ""},
		{"slow", "slow", true, nil, "warn"},
		{"all", "all", false, nil, "debug"},
		{"failure", "slow", false, &pgconn.PgError{Code: "23505", Message: "secret-error", Detail: "secret-detail"}, "error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := zerolog.New(&output).With().Str("request_id", "request-123").Logger()
			tracer := &queryLogger{mode: tc.mode, slow: time.Second}
			ctx := tracer.TraceQueryStart(logger.WithContext(context.Background()), nil, pgx.TraceQueryStartData{
				SQL: "-- name: FindAccount :one\nSELECT 'secret-sql'", Args: []any{"secret-argument"},
			})
			if tc.slow && tc.mode != "off" {
				trace := ctx.Value(queryTraceKey{}).(queryTrace)
				trace.started = time.Now().Add(-2 * time.Second)
				ctx = context.WithValue(ctx, queryTraceKey{}, trace)
			}
			tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{Err: tc.err, CommandTag: pgconn.NewCommandTag("SELECT 1")})
			if tc.level == "" {
				if output.Len() != 0 {
					t.Fatal("unwanted SQL log")
				}
				return
			}
			var event map[string]any
			if err := json.Unmarshal(output.Bytes(), &event); err != nil {
				t.Fatal(err)
			}
			if event["level"] != tc.level || event["query"] != "FindAccount" || event["component"] != "sql" || event["request_id"] != "request-123" {
				t.Fatalf("incorrect correlation: %v", event)
			}
			if tc.err != nil && event["sqlstate"] != "23505" {
				t.Fatal("missing diagnostic SQLSTATE")
			}
			if strings.Contains(output.String(), "secret-") {
				t.Fatal("SQL log leaked query/arguments/provider details")
			}
		})
	}
}

func TestQueryNameRejectsArbitrarySQL(t *testing.T) {
	for _, sql := range []string{"SELECT 'private'", "-- name: private@example.com :one\nSELECT 1", "-- name: x\nSELECT 1"} {
		if queryName(sql) != "unnamed" {
			t.Fatal("arbitrary SQL treated as a safe name")
		}
	}
}
