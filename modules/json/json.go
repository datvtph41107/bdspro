package json

import "io"

type Decoder interface {
	Decode(v any) error
}

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

func MarshalWriteRelaxed(w io.Writer, v any) error {
	return DefaultJSONHandler.MarshalWriteRelaxed(w, v)
}

func Unmarshal(data []byte, v any) error {
	return DefaultJSONHandler.Unmarshal(data, v)
}

func NewDecoder(reader io.Reader) Decoder {
	return DefaultJSONHandler.NewDecoder(reader)
}
