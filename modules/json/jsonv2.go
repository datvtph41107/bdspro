package json

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"io"
)

// JSONv2 owns the serialization policies shared by services/modules.
// It is deliberately stricter than encoding/json (v1) for untrusted input:
// invalid UTF-8 and duplicate object names are rejected by default.
type JSONv2 struct {
	marshalOptions                  jsonv2.Options
	marshalRelaxedOptions           jsonv2.Options
	marshalKeepOptionalEmptyOptions jsonv2.Options
	unmarshalOptions                jsonv2.Options
	unmarshalCaseInsensitiveOptions jsonv2.Options
}

func newJSONv2() *JSONv2 {
	commonMarshalOptions := jsonv2.JoinOptions(
		jsonv2.FormatNilSliceAsNull(true),
		jsonv2.FormatNilMapAsNull(true),
	)
	return &JSONv2{
		marshalOptions: commonMarshalOptions,
		marshalRelaxedOptions: jsonv2.JoinOptions(
			commonMarshalOptions, jsontext.AllowInvalidUTF8(true),
		),
		marshalKeepOptionalEmptyOptions: jsonv2.JoinOptions(
			commonMarshalOptions, jsonv1.OmitEmptyWithLegacySemantics(true),
		),
		unmarshalOptions: jsonv2.DefaultOptionsV2(),
		unmarshalCaseInsensitiveOptions: jsonv2.MatchCaseInsensitiveNames(true),
	}
}

var jsonV2 = newJSONv2()

func getDefaultJSONHandler() Interface {
	return jsonV2
}

// MarshalKeepOptionalEmpty retains legacy omitempty behavior for pointers and
// custom marshalers that produce an empty string.
func MarshalKeepOptionalEmpty(v any) ([]byte, error) {
	return jsonv2.Marshal(v, jsonV2.marshalKeepOptionalEmptyOptions)
}

// MarshalDeterministic is for reproducible JSON, not signature canonicalization.
// It does not define a cross-language canonical JSON scheme.
func MarshalDeterministic(v any) ([]byte, error) {
	return jsonv2.Marshal(v, jsonV2.marshalOptions, jsonv2.Deterministic(true))
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

func (j *JSONv2) NewEncoder(w io.Writer) Encoder {
	return &jsonV2Encoder{writer: w, opts: j.marshalOptions}
}

func (j *JSONv2) NewDecoder(r io.Reader) Decoder {
	return &jsonV2Decoder{reader: r, opts: j.unmarshalOptions}
}

func (*JSONv2) Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error {
	return jsonv1.Indent(dst, src, prefix, indent)
}

type jsonV2Encoder struct {
	writer io.Writer
	opts   jsonv2.Options
}

func (e *jsonV2Encoder) Encode(v any) error {
	return jsonv2.MarshalWrite(e.writer, v, e.opts)
}

type jsonV2Decoder struct {
	reader io.Reader
	opts   jsonv2.Options
}

func (d *jsonV2Decoder) Decode(v any) error {
	return jsonv2.UnmarshalRead(d.reader, v, d.opts)
}

// NewDecoderCaseInsensitive is an explicit escape hatch for legacy JSON
// sources. Never make case-insensitive decoding the default for auth input.
func NewDecoderCaseInsensitive(r io.Reader) Decoder {
	return &jsonV2Decoder{reader: r, opts: jsonV2.unmarshalCaseInsensitiveOptions}
}

type Value = jsontext.Value
