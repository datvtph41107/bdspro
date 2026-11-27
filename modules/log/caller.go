package log

import (
	"path"
	"runtime"
	"strconv"
	"strings"
)

const (
	projectPackage = "github.com/datvtph41107/bdspro/"
	loggingPackage = projectPackage + "modules/log."
	zerologPackage = "github.com/rs/zerolog"
)

// sourceLocation returns the first application frame outside the logging wrappers.
// CallersFrames accounts for inlined functions; no fixed wrapper depth is assumed.
func sourceLocation() (file, function string) {
	var pcs [32]uintptr
	n := runtime.Callers(2, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if !isLoggingFrame(frame.Function) {
			return applicationSource(frame)
		}
		if !more {
			return "", ""
		}
	}
}

func isLoggingFrame(function string) bool {
	return strings.HasPrefix(function, loggingPackage) ||
		strings.HasPrefix(function, zerologPackage+".") ||
		strings.HasPrefix(function, zerologPackage+"/")
}

// applicationSource uses the Go package path rather than leaking a build host's
// absolute filesystem path. External callers are not attributed to BDSPro.
func applicationSource(frame runtime.Frame) (file, function string) {
	if frame.File == "" || frame.Line <= 0 {
		return "", ""
	}
	base := path.Base(strings.ReplaceAll(frame.File, "\\", "/"))
	if name, ok := strings.CutPrefix(frame.Function, "main."); ok && base == "main.go" {
		return base + ":" + strconv.Itoa(frame.Line), name
	}
	name, ok := strings.CutPrefix(frame.Function, projectPackage)
	if !ok {
		return "", ""
	}
	start := strings.LastIndexByte(name, '/') + 1
	dot := strings.IndexByte(name[start:], '.')
	if dot < 0 {
		return "", ""
	}
	sep := start + dot
	pkg := name[:sep]
	if strings.HasSuffix(base, "_test.go") {
		pkg = strings.TrimSuffix(pkg, "_test")
	}
	file = path.Join(pkg, base)
	return file + ":" + strconv.Itoa(frame.Line), name[sep+1:]
}
