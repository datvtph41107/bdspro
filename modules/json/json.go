// Package json centralizes the encoding/json/v2 policy used by BDSPro.
// Public functions exist only when production code has a concrete caller.
package json

import "io"

// Decoder holds one JSON stream reader. A second backend is not required.
type Decoder struct {
	reader io.Reader
}

// Marshal preserves the established v1 null representation of nil slices/maps.
func Marshal(value any) ([]byte, error) { return marshalV2(value) }

// Unmarshal uses strict JSON v2 semantics, rejecting duplicate object names
// and invalid UTF-8 rather than silently accepting ambiguous input.
func Unmarshal(data []byte, target any) error { return unmarshalV2(data, target) }

// NewDecoder supports provider responses without buffering them in the caller.
func NewDecoder(reader io.Reader) *Decoder { return &Decoder{reader: reader} }
