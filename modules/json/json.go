// Package json centralizes JSON policy. It uses the standard Go 1.27 JSON v2
// implementation and exposes consistent marshal/stream/format operations.
// See jsonv2.go for explicit compatibility and security semantics.
package json

import (
	"bytes"
	jsonv1 "encoding/json"
	"io"
)

type Encoder interface {
	Encode(v any) error
}

type Decoder interface {
	Decode(v any) error
}

// Interface makes the JSON implementation replaceable without forcing every
// consumer to couple to the implementation's encoder/decoder concrete types.
type Interface interface {
	Marshal(v any) ([]byte, error)
	MarshalWriteRelaxed(w io.Writer, v any) error
	Unmarshal(data []byte, v any) error
	NewEncoder(w io.Writer) Encoder
	NewDecoder(r io.Reader) Decoder
	Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error
}

// DefaultJSONHandler is chosen once per process. Treat it as immutable after
// startup; changing it while other goroutines use JSON would be a data race.
var DefaultJSONHandler Interface = getDefaultJSONHandler()

func Marshal(v any) ([]byte, error) {
	return DefaultJSONHandler.Marshal(v)
}

func MarshalWriteRelaxed(w io.Writer, v any) error {
	return DefaultJSONHandler.MarshalWriteRelaxed(w, v)
}

func Unmarshal(data []byte, v any) error {
	return DefaultJSONHandler.Unmarshal(data, v)
}

func NewEncoder(w io.Writer) Encoder {
	return DefaultJSONHandler.NewEncoder(w)
}

func NewDecoder(r io.Reader) Decoder {
	return DefaultJSONHandler.NewDecoder(r)
}

func Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error {
	return DefaultJSONHandler.Indent(dst, src, prefix, indent)
}

func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	src, err := Marshal(v)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := Indent(&buf, src, prefix, indent); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Valid(data []byte) bool {
	return jsonv1.Valid(data)
}
