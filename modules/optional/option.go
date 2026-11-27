package optional

import (
	"bytes"
	"encoding/json"
	"errors"
)

// Option distinguishes an omitted PATCH field from a supplied value, including
// an empty string. JSON null is deliberately rejected by UnmarshalJSON.
type Option[T any] []T

func None[T any]() Option[T]        { return nil }
func Some[T any](value T) Option[T] { return Option[T]{value} }

func (o Option[T]) Has() bool { return len(o) != 0 }

func (o Option[T]) Value() T {
	if o.Has() {
		return o[0]
	}
	var zero T
	return zero
}

func (o *Option[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("null is not a valid optional field value")
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*o = Some(value)
	return nil
}

func (o Option[T]) MarshalJSON() ([]byte, error) {
	if !o.Has() {
		return []byte("null"), nil
	}
	return json.Marshal(o.Value())
}
