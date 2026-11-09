// Package json owns the application JSON codec, as in Gitea.
// The encoding/json/v2 implementation lives in jsonv2.go.
package json

import "io"

// Decoder is the common JSON decode contract.
type Decoder interface {
	Decode(v any) error
}

// Interface is the one codec contract used by persistence, providers and HTTP.
type Interface interface {
	Marshal(v any) ([]byte, error)
	MarshalWriteRelaxed(w io.Writer, v any) error
	Unmarshal(data []byte, v any) error
	NewDecoder(reader io.Reader) Decoder
}

var DefaultJSONHandler Interface = getDefaultJSONHandler()

func Marshal(v any) ([]byte, error) {
	return DefaultJSONHandler.Marshal(v)
}

// MarshalWriteRelaxed tolerates invalid UTF-8 in outgoing text, following
// Gitea's HTTP response policy. Strict input decoding is unchanged.
func MarshalWriteRelaxed(w io.Writer, v any) error {
	return DefaultJSONHandler.MarshalWriteRelaxed(w, v)
}

func Unmarshal(data []byte, v any) error {
	return DefaultJSONHandler.Unmarshal(data, v)
}

func NewDecoder(reader io.Reader) Decoder {
	return DefaultJSONHandler.NewDecoder(reader)
}
