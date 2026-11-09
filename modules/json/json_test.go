package json

import (
	"bytes"
	legacyjson "encoding/json"
	"strings"
	"testing"
)

func TestMarshalPreservesPersistedNilCollectionRepresentation(t *testing.T) {
	type value struct {
		Names  []string       `json:"names"`
		Counts map[string]int `json:"counts"`
	}
	original, err := legacyjson.Marshal(value{})
	if err != nil {
		t.Fatal(err)
	}
	current, err := Marshal(value{})
	if err != nil || !bytes.Equal(original, current) {
		t.Fatalf("old and new JSON representation diverged: old=%s new=%s err=%v", original, current, err)
	}
}

func TestUnmarshalRejectsAmbiguousJSON(t *testing.T) {
	var target struct {
		Name string `json:"name"`
	}
	if err := Unmarshal([]byte(`{"name":"one","name":"two"}`), &target); err == nil {
		t.Fatal("duplicate JSON properties must be rejected")
	}
	target.Name = ""
	if err := Unmarshal([]byte{'{', '"', 'n', 'a', 'm', 'e', '"', ':', '"', 0xff, '"', '}'}, &target); err == nil {
		t.Fatal("invalid UTF-8 must be rejected")
	}
	target.Name = ""
	if err := Unmarshal([]byte(`{"NAME":"alice"}`), &target); err != nil || target.Name != "" {
		t.Fatalf("JSON property matching became case-insensitive: %+v err=%v", target, err)
	}
}

func TestNewDecoderReadsBoundedProviderResponse(t *testing.T) {
	var result struct {
		Token string `json:"id_token"`
	}
	if err := NewDecoder(strings.NewReader(`{"id_token":"signed"}`)).Decode(&result); err != nil || result.Token != "signed" {
		t.Fatalf("stream decoder failed: %+v err=%v", result, err)
	}
}

// HTTP output uses the relaxed Gitea codec while persisted payloads retain
// the strict Marshal/Unmarshal contract verified above.
func TestMarshalWriteRelaxedReplacesInvalidUTF8(t *testing.T) {
	var output bytes.Buffer
	if err := MarshalWriteRelaxed(&output, map[string]string{"value": string([]byte{0xff})}); err != nil {
		t.Fatal(err)
	}
	var value struct {
		Value string `json:"value"`
	}
	if err := legacyjson.Unmarshal(output.Bytes(), &value); err != nil || value.Value != "\ufffd" {
		t.Fatalf("invalid UTF-8 response=%s value=%q err=%v", output.Bytes(), value.Value, err)
	}
}
