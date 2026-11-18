package websocket

import (
	"bytes"
	"errors"
	"testing"
)

func TestEncodeEventBroadcast(t *testing.T) {
	data, err := EncodeEvent(
		"notification-count",
		"",
		struct {
			Count int64 `json:"count"`
		}{Count: 3},
	)
	if err != nil {
		t.Fatal(err)
	}

	got, ok := EventForSession(data, "session-A")
	if !ok {
		t.Fatal("broadcast event rejected")
	}

	want := []byte(
		`{"eventType":"notification-count","eventData":{"count":3}}`,
	)
	if !bytes.Equal(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestEncodeEventSessionTarget(t *testing.T) {
	data, err := EncodeEvent(
		"session-refresh",
		"session-A",
		map[string]bool{"refresh": true},
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := EventForSession(data, "session-B"); ok {
		t.Fatal("event leaked to another session")
	}

	if _, ok := EventForSession(data, "session-A"); !ok {
		t.Fatal("target session did not receive event")
	}
}

func TestEventForSessionRejectsMalformed(t *testing.T) {
	cases := [][]byte{
		[]byte("missing separator"),
		[]byte("\n"),
		[]byte("\nnot-json"),
	}

	for _, data := range cases {
		if _, ok := EventForSession(data, "session-A"); ok {
			t.Fatalf("accepted malformed event %q", data)
		}
	}
}

func TestEncodeEventRejectsInvalidInput(t *testing.T) {
	_, err := EncodeEvent("", "", nil)
	if !errors.Is(err, ErrEventType) {
		t.Fatalf("expected event type error, got %v", err)
	}

	_, err = EncodeEvent("test", "session\nother", nil)
	if !errors.Is(err, ErrSessionTarget) {
		t.Fatalf("expected session target error, got %v", err)
	}
}
