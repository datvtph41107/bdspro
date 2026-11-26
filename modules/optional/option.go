package optional

import (
	"bytes"
	"encoding/json"
	"errors"
)

type Option[T any] []T

func None[T any]() Option[T]        { return nil }
func Some[T any](value T) Option[T] { return Option[T]{value} }

func FromPtr[T any](ptr *T) Option[T] {
	if ptr == nil {
		return None[T]()
	}
	return Some(*ptr)
}

func (o Option[T]) Has() bool { return len(o) != 0 }

func (o Option[T]) Value() T {
	var zero T
	return o.ValueOrDefault(zero)
}

func (o Option[T]) ValueOrDefault(fallback T) T {
	if o.Has() {
		return o[0]
	}
	return fallback
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
