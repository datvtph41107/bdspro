package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestJSONBinder(t *testing.T) {
	type option struct { Name string `json:"name"` }
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Backend"}`))
	form, err := BindJSON[option](RequestJSONBinder{Request: req})
	if err != nil || form.Name != "Backend" {
		t.Fatalf("form=%+v err=%v", form, err)
	}
}

func TestRequestJSONBinderRejectsTrailingDocument(t *testing.T) {
	type option struct { Name string `json:"name"` }
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"A"} {"name":"B"}`))
	_, err := BindJSON[option](RequestJSONBinder{Request: req})
	if err == nil {
		t.Fatal("must reject trailing JSON document")
	}
}
