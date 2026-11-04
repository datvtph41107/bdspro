// Package json owns the application's JSON v2 wire policy.
// Persisted OAuth state and queued task payloads retain their existing shape.
package json

import (
	jsonv2 "encoding/json/v2"
	"io"
)

var (
	marshalOptions = jsonv2.JoinOptions(
		jsonv2.FormatNilSliceAsNull(true),
		jsonv2.FormatNilMapAsNull(true),
	)
	unmarshalOptions = jsonv2.DefaultOptionsV2()
)

// Marshal retains legacy nil collection encoding for persisted payloads.
func Marshal(value any) ([]byte, error) {
	return jsonv2.Marshal(value, marshalOptions)
}

// Unmarshal applies strict JSON v2 decoding (including duplicate-name rejection).
func Unmarshal(data []byte, target any) error {
	return jsonv2.Unmarshal(data, target, unmarshalOptions)
}

// Decoder decodes a single JSON value from a reader.
type Decoder struct {
	reader io.Reader
}

// NewDecoder is used for bounded provider responses.
func NewDecoder(reader io.Reader) *Decoder {
	return &Decoder{reader: reader}
}

func (d *Decoder) Decode(target any) error {
	return jsonv2.UnmarshalRead(d.reader, target, unmarshalOptions)
}
