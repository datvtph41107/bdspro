package json

import (
	"bytes"
	"strings"
	"testing"
)

func TestBasicMarshalAndUnmarshal(t *testing.T) {
	type doc struct {
		Values []string       `json:"values"`
		Labels map[string]int `json:"labels"`
	}
	raw, err := Marshal(doc{})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"values":null,"labels":null}` {
		t.Fatalf("unexpected nil semantics: %s", raw)
	}
	var got doc
	if err := Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
}

func TestStrictDecodeRejectsDuplicateNamesAndInvalidUTF8(t *testing.T) {
	var target struct {
		Name string `json:"name"`
	}
	if err := Unmarshal([]byte(`{"name":"one","name":"two"}`), &target); err == nil {
		t.Fatal("duplicate JSON member accepted")
	}
	if err := Unmarshal([]byte{'{', '"', 'n', 'a', 'm', 'e', '"', ':', '"', 0xff, '"', '}'} , &target); err == nil {
		t.Fatal("invalid UTF-8 accepted")
	}
	if err := Unmarshal([]byte(`{"NAME":"Alice"}`), &target); err != nil {
		t.Fatal(err)
	} else if target.Name != "" {
		t.Fatal("case-insensitive matching applied by default")
	}
	if err := NewDecoderCaseInsensitive(strings.NewReader(`{"NAME":"Alice"}`)).Decode(&target); err != nil || target.Name != "Alice" {
		t.Fatalf("case-insensitive opt-in failed: %v %+v", err, target)
	}
}

func TestEncoderDecoderAndRelaxedInvalidUTF8(t *testing.T) {
	var buffer bytes.Buffer
	if err := NewEncoder(&buffer).Encode(map[string]any{"x": 4}); err != nil {
		t.Fatal(err)
	}
	var result map[string]int
	if err := NewDecoder(&buffer).Decode(&result); err != nil || result["x"] != 4 {
		t.Fatalf("JSON stream mismatch: %+v err=%v", result, err)
	}
	buffer.Reset()
	if err := MarshalWriteRelaxed(&buffer, map[string]string{"text": string([]byte{0xff})}); err != nil {
		t.Fatal(err)
	}
	var relaxed map[string]string
	if err := Unmarshal(buffer.Bytes(), &relaxed); err != nil || relaxed["text"] != "\ufffd" {
		t.Fatalf("relaxed UTF-8 must replace invalid code units: %q err=%v", relaxed["text"], err)
	}
}

func TestDeterministicAndOptionalEmpty(t *testing.T) {
	raw, err := MarshalDeterministic(map[string]int{"z": 2, "a": 1})
	if err != nil || string(raw) != `{"a":1,"z":2}` {
		t.Fatalf("unstable deterministic output %s: %v", raw, err)
	}
	empty := ""
	value := struct {
		Name *string `json:"name,omitempty"`
	}{Name: &empty}
	keep, err := MarshalKeepOptionalEmpty(value)
	if err != nil || string(keep) != `{"name":""}` {
		t.Fatalf("legacy optional-empty semantics not retained: %s %v", keep, err)
	}
}

func TestValidIndentAndMarshalIndent(t *testing.T) {
	if !Valid([]byte(`{"x":1}`)) || Valid([]byte(`{"x":`)) {
		t.Fatal("JSON validity check incorrect")
	}
	formatted, err := MarshalIndent(map[string]int{"x": 1}, "", "  ")
	if err != nil || !bytes.Contains(formatted, []byte("\n  \"x\": 1")) {
		t.Fatalf("MarshalIndent failed: %s %v", formatted, err)
	}
	var b bytes.Buffer
	if err := Indent(&b, []byte(`{"x":1}`), ">", "  "); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), ">  \"x\": 1") {
		t.Fatalf("Indent prefix not applied: %s", b.String())
	}
}
