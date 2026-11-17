// Package optional distinguishes an omitted value from an explicit zero.
package optional

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
