package util

// WithOperation annotates an error with a static, developer-owned operation ID.
// Never pass request data, credentials or raw error messages as operation.
func WithOperation(operation string, err error) error {
	if err == nil {
		return nil
	}
	return operationError{operation: operation, cause: err}
}

type operationError struct {
	operation string
	cause     error
}

func (e operationError) Error() string { return e.cause.Error() }
func (e operationError) Unwrap() error { return e.cause }

// ErrorOperation examines only this error node; callers walk Unwrap separately.
func ErrorOperation(err error) string {
	if e, ok := err.(operationError); ok {
		return e.operation
	}
	return ""
}
