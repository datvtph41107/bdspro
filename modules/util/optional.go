package util

func OptionalArg[T any](args []T, fallback ...T) (result T) {
	if len(args) > 0 {
		return args[0]
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return result
}
