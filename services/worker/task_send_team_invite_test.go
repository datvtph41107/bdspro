package worker

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

func TestTeamInviteTaskRejectsInvalidPayload(t *testing.T) {
	handler := NewTeamInviteHandler(nil, nil, "01234567890123456789012345678901")
	for _, payload := range [][]byte{[]byte("{"), []byte("{}")} {
		if err := handler.ProcessTask(context.Background(), asynq.NewTask(TaskSendTeamInvite, payload)); err == nil {
			t.Fatal("invalid invitation task payload was accepted")
		}
	}
	id := uuid.New()
	body, err := json.Marshal(PayloadSendTeamInvite{InviteID: id})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "token") || !strings.Contains(string(body), id.String()) {
		t.Fatal("task must contain only the invitation ID")
	}
}
