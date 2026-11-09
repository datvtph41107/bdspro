package web

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/reqctx"
)

type testFormStore struct{ data reqctx.ContextData }

func (s *testFormStore) GetData() reqctx.ContextData { return s.data }

func TestFormRequestIsolation(t *testing.T) {
	a := &testFormStore{data: make(reqctx.ContextData)}
	b := &testFormStore{data: make(reqctx.ContextData)}
	SetForm(a, &struct{ Name string }{Name: "a"})
	SetForm(b, &struct{ Name string }{Name: "b"})
	if v := GetForm[*struct{ Name string }](a); v.Name != "a" {
		t.Fatalf("A: %q", v.Name)
	}
	if v := GetForm[*struct{ Name string }](b); v.Name != "b" {
		t.Fatalf("B: %q", v.Name)
	}
}

func TestFormTypeMismatchPanics(t *testing.T) {
	s := &testFormStore{data: make(reqctx.ContextData)}
	SetForm(s, &struct{ Name string }{})
	defer func() {
		if recover() == nil {
			t.Fatal("missing type mismatch panic")
		}
	}()
	_ = GetForm[*struct{ ID int64 }](s)
}

type testJSONFormContext struct {
	testFormStore
	body string
	err  error
}

func (c *testJSONFormContext) BindJSON(value any) error {
	if c.err != nil {
		return c.err
	}
	return json.NewDecoder(strings.NewReader(c.body)).Decode(value)
}

func TestBindJSONStoresTypedFormOnlyAfterSuccess(t *testing.T) {
	type teamForm struct{ Name string }
	valid := &testJSONFormContext{
		testFormStore: testFormStore{data: make(reqctx.ContextData)},
		body:          `{"name":"Backend"}`,
	}
	form, err := BindJSON[teamForm](valid)
	if err != nil || form.Name != "Backend" {
		t.Fatalf("form=%+v err=%v", form, err)
	}
	if got := GetForm[*teamForm](valid); got != form {
		t.Fatal("form was copied or stored under a second key")
	}
	bad := &testJSONFormContext{
		testFormStore: testFormStore{data: make(reqctx.ContextData)},
		err:           errors.New("invalid json"),
	}
	if _, err := BindJSON[teamForm](bad); err == nil {
		t.Fatal("invalid JSON unexpectedly accepted")
	}
	if _, ok := bad.data[formKey]; ok {
		t.Fatal("rejected form leaked into request data")
	}
}
