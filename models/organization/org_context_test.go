package organization

import (
	"context"
	"errors"
	"testing"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
)

func TestListForDoerRequiresActingPrincipal(t *testing.T) {
	_, err := ListForDoer(context.Background())
	if !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("expected unverified principal error, got %v", err)
	}
}
