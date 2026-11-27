package log_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
)

func TestErrorDetailsPrivacyAndStack(t *testing.T) {
	var out bytes.Buffer
	logger := zerolog.New(&out)
	cause := &pgconn.PgError{Code: "23505", Message: "secret-message", Detail: "secret-detail"}
	err := util.WithOperation("auth.session.create", cause)
	for _, stack := range []bool{false, true} {
		out.Reset()
		log.ErrorDetails(logger.Error(), err, stack).Msg("http.application_error")
		var event map[string]any
		if err := json.Unmarshal(out.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.String(), "secret-") || !strings.Contains(out.String(), "23505") {
			t.Fatalf("unsafe or incomplete event: %s", out.String())
		}
		_, hasStack := event["stack_frames"]
		if hasStack != stack {
			t.Fatalf("unexpected stack: %v", event)
		}
	}
	if !errors.Is(err, cause) {
		t.Fatal("lost wrapped cause")
	}
}
