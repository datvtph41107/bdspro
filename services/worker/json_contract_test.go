package worker

import (
	"bytes"
	legacyjson "encoding/json"
	"testing"

	appjson "github.com/datvtph41107/bdspro/modules/json"
	"github.com/google/uuid"
)

// Existing PostgreSQL outbox records and Redis Asynq tasks may have been
// serialized by encoding/json v1. The R29 decoder must still read them.
func TestPersistedTaskPayloadCompatibility(t *testing.T) {
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	cases := []struct {
		name  string
		value any
	}{
		{"verification", PayloadSendVerifyEmail{AccountID: 81}},
		{"password reset", PayloadSendPasswordReset{AccountID: 81}},
		{"team invite", PayloadSendTeamInvite{InviteID: id}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			oldPayload, err := legacyjson.Marshal(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			newPayload, err := appjson.Marshal(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(oldPayload, newPayload) {
				t.Fatalf("persisted task payload changed: old=%s new=%s", oldPayload, newPayload)
			}
			var got map[string]any
			if err := appjson.Unmarshal(oldPayload, &got); err != nil {
				t.Fatalf("old payload rejected: %v", err)
			}
			if len(got) == 0 {
				t.Fatal("decoded payload unexpectedly empty")
			}
		})
	}
}

func TestTaskPayloadRejectsDuplicateMembers(t *testing.T) {
	var payload PayloadSendVerifyEmail
	if err := appjson.Unmarshal([]byte(`{"account_id":2,"account_id":3}`), &payload); err == nil {
		t.Fatal("duplicate payload properties accepted")
	}
}
