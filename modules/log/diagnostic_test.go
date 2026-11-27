package log

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/util"
)

type sampleDBError struct{}

func (sampleDBError) Error() string    { return "secret-password" }
func (sampleDBError) SQLState() string { return "23505" }

type cyclicError struct{}

func (e cyclicError) Error() string { return "secret-cycle" }
func (e cyclicError) Unwrap() error { return e }

func TestDescribeError(t *testing.T) {
	base := sampleDBError{}
	err := util.WithOperation("auth.login", util.WithOperation("auth.session.create", base))
	if !errors.Is(err, base) || err.Error() != base.Error() {
		t.Fatal("operation annotation changed error semantics")
	}
	nodes, truncated := describeError(errors.Join(err, fmt.Errorf("secret-token: %w", errors.New("secret-value"))))
	if truncated || len(nodes) != 6 || nodes[1].Operation != "auth.login" || nodes[2].Operation != "auth.session.create" || nodes[3].SQLState != "23505" || nodes[4].Depth != 1 {
		t.Fatalf("unexpected chain: %+v", nodes)
	}
	encoded, marshalErr := json.Marshal(nodes)
	if marshalErr != nil || strings.Contains(string(encoded), "secret-") {
		t.Fatalf("unsafe chain: %s, %v", encoded, marshalErr)
	}
	t.Log(string(encoded))
}

func TestDescribeErrorBounded(t *testing.T) {
	nodes, truncated := describeError(cyclicError{})
	if !truncated || len(nodes) != diagnosticLimit {
		t.Fatalf("unexpected limit: %d %t", len(nodes), truncated)
	}
	if nodes, truncated := describeError(nil); len(nodes) != 0 || truncated {
		t.Fatal("nil error generated diagnostics")
	}
	if validSQLState("abcde") || validSQLState("123456") || validSQLState("12\n45") || !validSQLState("23505") {
		t.Fatal("SQLSTATE validation failed")
	}
}
