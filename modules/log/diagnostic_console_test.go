package log

import (
	"bytes"
	"strings"
	"testing"
)

func TestDiagnosticConsolePreservesJSON(t *testing.T) {
	raw := []byte(`{"level":"error","message":"http.application_error","error_type":"util.operationError","error_chain":[{"type":"util.operationError","depth":0,"operation":"auth.session.create"},{"type":"*pgconn.PgError","depth":1,"sqlstate":"23505"}],"stack_frames":["services/context/api.go:82:APIErrorAuto"]}` + "\n")
	before := bytes.Clone(raw)
	var terminal, file bytes.Buffer
	if _, err := (fanoutWriter{newConsoleWriter(&terminal, true, true), &file}).Write(raw); err != nil {
		t.Fatal(err)
	}
	line := terminal.String()
	for _, want := range []string{"auth.session.create", "sqlstate=23505"} {
		if !strings.Contains(line, want) {
			t.Fatalf("missing %q in %s", want, line)
		}
	}
	for _, hide := range []string{"stack_frames=", "error_chain=", "services/context/api.go"} {
		if strings.Contains(line, hide) {
			t.Fatalf("verbose field %q in %s", hide, line)
		}
	}
	if !bytes.Equal(before, file.Bytes()) || !bytes.Equal(before, raw) {
		t.Fatal("console projection altered JSON")
	}
}
