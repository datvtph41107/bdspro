package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// queryLogger observes the existing pgx boundary; services do not log SQL.
// SQL text, parameters and PostgreSQL error details may contain credentials.
type queryLogger struct {
	mode string
	slow time.Duration
}
type queryTraceKey struct{}
type queryTrace struct {
	started           time.Time
	name, fingerprint string
}

func (l *queryLogger) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if l.mode == "off" {
		return ctx
	}
	digest := sha256.Sum256([]byte(data.SQL))
	trace := queryTrace{started: time.Now(), name: queryName(data.SQL), fingerprint: hex.EncodeToString(digest[:6])}
	return context.WithValue(ctx, queryTraceKey{}, trace)
}

func (l *queryLogger) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	trace, ok := ctx.Value(queryTraceKey{}).(queryTrace)
	if !ok {
		return
	}
	elapsed := time.Since(trace.started)
	if data.Err == nil && elapsed < l.slow && l.mode != "all" {
		return
	}
	logger := log.For(ctx, "sql")
	event := logger.Debug()
	if data.Err != nil {
		event = logger.Error()
	} else if elapsed >= l.slow {
		event = logger.Warn()
	}
	event.Str("query", trace.name).Str("query_id", trace.fingerprint).
		Dur("duration_ms", elapsed).Int64("rows", data.CommandTag.RowsAffected())
	if data.Err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.As(data.Err, &pgErr):
			event.Str("sqlstate", pgErr.Code).Str("reason", "postgres_error")
		case errors.Is(data.Err, context.Canceled):
			event.Str("reason", "query_canceled")
		case errors.Is(data.Err, context.DeadlineExceeded):
			event.Str("reason", "query_timeout")
		default:
			event.Str("reason", "query_failed")
		}
	}
	event.Msg("db.query")
}

// Only SQLC's identifier comment is printable. Unnamed SQL gets a fingerprint.
func queryName(sql string) string {
	line, _, _ := strings.Cut(sql, "\n")
	fields := strings.Fields(line)
	if len(fields) != 4 || fields[0] != "--" || fields[1] != "name:" {
		return "unnamed"
	}
	name := fields[2]
	if len(name) == 0 || len(name) > 100 {
		return "unnamed"
	}
	for _, c := range name {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return "unnamed"
		}
	}
	return name
}
