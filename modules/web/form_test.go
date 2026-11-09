package web

import (
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
