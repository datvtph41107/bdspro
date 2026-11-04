package json

import (
	jsonv2 "encoding/json/v2"
)

// These options preserve the JSON representation used by pre-R30 jobs and
// OAuth2 state. Do not silently enable case-insensitive names or relaxed UTF-8.
var marshalOptions = jsonv2.JoinOptions(
	jsonv2.FormatNilSliceAsNull(true),
	jsonv2.FormatNilMapAsNull(true),
)

func marshalV2(value any) ([]byte, error) {
	return jsonv2.Marshal(value, marshalOptions)
}

func unmarshalV2(data []byte, target any) error {
	return jsonv2.Unmarshal(data, target, jsonv2.DefaultOptionsV2())
}

func (d *Decoder) Decode(target any) error {
	return jsonv2.UnmarshalRead(d.reader, target, jsonv2.DefaultOptionsV2())
}
