package middleware

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type jsonBinderStub struct {
	body     string
	err      error
	received any
}

func (b *jsonBinderStub) BindJSON(value any) error {
	b.received = value
	if b.err != nil {
		return b.err
	}
	return json.NewDecoder(strings.NewReader(b.body)).Decode(value)
}

func TestBindJSONUsesRequestBinderAndReturnsForm(t *testing.T) {
	type teamOption struct {
		Name string `json:"name"`
	}
	binder := &jsonBinderStub{body: `{"name":"Backend"}`}
	form, err := BindJSON[teamOption](binder)
	if err != nil || form.Name != "Backend" || binder.received != form {
		t.Fatalf("form=%+v, err=%v, received=%T", form, err, binder.received)
	}
}

func TestBindJSONReturnsBinderErrorWithoutHidingIt(t *testing.T) {
	type teamOption struct{ Name string }
	sentinel := errors.New("binding failed")
	binder := &jsonBinderStub{err: sentinel}
	form, err := BindJSON[teamOption](binder)
	if !errors.Is(err, sentinel) || form == nil || binder.received != form {
		t.Fatalf("form=%+v err=%v received=%T", form, err, binder.received)
	}
}

func TestBindJSONRejectsNonStructTemplate(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("a scalar template must fail instead of silently being bound")
		}
	}()
	_, _ = BindJSON[string](&jsonBinderStub{body: `"scalar"`})
}
