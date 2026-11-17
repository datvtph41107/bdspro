package util

// OptionalArg picks the first optional argument, then the first fallback,
// otherwise the zero value. Only the first element of each slice is used.
func OptionalArg[T any](args []T, fallback ...T) (result T) {
	if len(args) > 0 {
		return args[0]
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return result
}
