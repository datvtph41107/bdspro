package middleware

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type jsonBinderTest struct {
	body string
	err  error
}

func (b jsonBinderTest) BindJSON(value any) error {
	if b.err != nil {
		return b.err
	}
	return json.NewDecoder(strings.NewReader(b.body)).Decode(value)
}

func TestBindJSONTypedForm(t *testing.T) {
	type form struct {
		Name string `json:"name"`
	}
	got, err := BindJSON[form](jsonBinderTest{body: `{"name":"team"}`})
	if err != nil || got.Name != "team" {
		t.Fatalf("form=%+v err=%v", got, err)
	}
	failure := errors.New("binding rejected")
	got, err = BindJSON[form](jsonBinderTest{err: failure})
	if !errors.Is(err, failure) || got == nil {
		t.Fatalf("binding error=%v form=%+v", err, got)
	}
}
