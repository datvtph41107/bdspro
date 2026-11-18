package websocket

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrEventType     = errors.New("websocket: invalid event type")
	ErrSessionTarget = errors.New("websocket: invalid session target")
	ErrEventEnvelope = errors.New("websocket: invalid event envelope")
)

// EncodeEvent creates an internal broker envelope:
//
//	targetSession + "\n" + JSON event
//
// An empty targetSession means every connection
// subscribed to the topic may receive the event.
func EncodeEvent(
	eventType string,
	targetSession string,
	eventData any,
) ([]byte, error) {
	if strings.TrimSpace(eventType) == "" {
		return nil, ErrEventType
	}
	if strings.ContainsAny(targetSession, "\r\n") {
		return nil, ErrSessionTarget
	}

	payload, err := json.Marshal(struct {
		EventType string `json:"eventType"`
		EventData any    `json:"eventData"`
	}{
		EventType: eventType,
		EventData: eventData,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"websocket: encode event: %w", err,
		)
	}

	result := make(
		[]byte,
		0,
		len(targetSession)+1+len(payload),
	)
	result = append(result, targetSession...)
	result = append(result, '\n')
	result = append(result, payload...)

	return result, nil
}

// EventForSession extracts the client JSON from
// a broker envelope.
//
// accepted=false means this event must not be sent.
// It may be targeted elsewhere or malformed.
func EventForSession(
	brokerPayload []byte,
	sessionID string,
) (eventData []byte, accepted bool) {
	target, payload, found := bytes.Cut(
		brokerPayload,
		[]byte("\n"),
	)
	if !found || len(payload) == 0 {
		return nil, false
	}

	if len(target) > 0 &&
		string(target) != sessionID {
		return nil, false
	}

	if !json.Valid(payload) {
		return nil, false
	}

	return payload, true
}
