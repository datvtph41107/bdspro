package log

import (
	"fmt"
	"runtime"

	"github.com/datvtph41107/bdspro/modules/util"
)

const diagnosticLimit = 12

// ErrorNode contains only diagnostic metadata, never the text of an error.
// Depth preserves the shape of joined errors in the pre-order traversal.
type ErrorNode struct {
	Type      string `json:"type"`
	Depth     int    `json:"depth"`
	Operation string `json:"operation,omitempty"`
	SQLState  string `json:"sqlstate,omitempty"`
}

func describeError(err error) (nodes []ErrorNode, truncated bool) {
	var walk func(error, int)
	walk = func(err error, depth int) {
		if err == nil {
			return
		}
		if len(nodes) >= diagnosticLimit {
			truncated = true
			return
		}
		node := ErrorNode{Type: fmt.Sprintf("%T", err), Depth: depth, Operation: util.ErrorOperation(err)}
		if state, ok := err.(interface{ SQLState() string }); ok {
			if code := state.SQLState(); validSQLState(code) {
				node.SQLState = code
			}
		}
		nodes = append(nodes, node)
		switch e := err.(type) {
		case interface{ Unwrap() []error }:
			for _, child := range e.Unwrap() {
				walk(child, depth+1)
				if truncated {
					break
				}
			}
		case interface{ Unwrap() error }:
			walk(e.Unwrap(), depth+1)
		}
	}
	walk(err, 0)
	return nodes, truncated
}

func validSQLState(code string) bool {
	if len(code) != 5 {
		return false
	}
	for _, c := range code {
		if !(c >= '0' && c <= '9' || c >= 'A' && c <= 'Z') {
			return false
		}
	}
	return true
}

// applicationStack records active application frames, not the origin of an error
// that has already been returned. No arguments or absolute filesystem paths.
func applicationStack() []string {
	var pcs [64]uintptr
	n := runtime.Callers(2, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	var stack []string
	for {
		frame, more := frames.Next()
		if !isLoggingFrame(frame.Function) {
			if file, function := applicationSource(frame); file != "" {
				stack = append(stack, file+":"+function)
			}
		}
		if !more || len(stack) >= diagnosticLimit {
			return stack
		}
	}
}
