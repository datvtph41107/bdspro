package json

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"io"
)

// JSONv2 is the concrete JSON codec; consumers use the facade in json.go.
type JSONv2 struct {
	marshalOptions        jsonv2.Options
	marshalRelaxedOptions jsonv2.Options
	unmarshalOptions      jsonv2.Options
}

var jsonV2 JSONv2

func init() {
	jsonV2.marshalOptions = jsonv2.JoinOptions(
		jsonv2.FormatNilSliceAsNull(true),
		jsonv2.FormatNilMapAsNull(true),
	)
	jsonV2.marshalRelaxedOptions = jsonv2.JoinOptions(
		jsonV2.marshalOptions,
		jsontext.AllowInvalidUTF8(true),
	)
	jsonV2.unmarshalOptions = jsonv2.DefaultOptionsV2()
}

func getDefaultJSONHandler() Interface {
	return &jsonV2
}

func (j *JSONv2) Marshal(v any) ([]byte, error) {
	return jsonv2.Marshal(v, j.marshalOptions)
}

func (j *JSONv2) MarshalWriteRelaxed(w io.Writer, v any) error {
	return jsonv2.MarshalWrite(w, v, j.marshalRelaxedOptions)
}

func (j *JSONv2) Unmarshal(data []byte, v any) error {
	return jsonv2.Unmarshal(data, v, j.unmarshalOptions)
}

func (j *JSONv2) NewDecoder(reader io.Reader) Decoder {
	return &jsonV2Decoder{reader: reader, opts: j.unmarshalOptions}
}

type jsonV2Decoder struct {
	reader io.Reader
	opts   jsonv2.Options
}

func (d *jsonV2Decoder) Decode(v any) error {
	return jsonv2.UnmarshalRead(d.reader, v, d.opts)
}
