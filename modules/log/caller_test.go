package log

import (
	"runtime"
	"testing"
)

func TestApplicationSource(t *testing.T) {
	for _, tt := range []struct {
		name, function, filename string
		line                     int
		wantFile, wantFunc       string
	}{
		{"normal", projectPackage + "services/auth.Login", "/home/dev/bdspro/services/auth/session.go", 84, "services/auth/session.go:84", "Login"},
		{"method", projectPackage + "models/db.(*queryLogger).TraceQueryEnd", "/src/models/db/query_log.go", 68, "models/db/query_log.go:68", "(*queryLogger).TraceQueryEnd"},
		{"windows", projectPackage + "services/auth.Login", `C:\dev\bdspro\services\auth\session.go`, 84, "services/auth/session.go:84", "Login"},
		{"trimpath", projectPackage + "routers.requestLogging.func1", projectPackage + "routers/request_log.go", 47, "routers/request_log.go:47", "requestLogging.func1"},
		{"external test package", projectPackage + "modules/log_test.TestCaller", "/src/modules/log/caller_test.go", 12, "modules/log/caller_test.go:12", "TestCaller"},
		{"main", "main.main", "/src/main.go", 12, "main.go:12", "main"},
		{"external", "net/http.(*Server).Serve", "/usr/local/go/src/net/http/server.go", 1, "", ""},
		{"invalid", projectPackage + "services/auth.Login", "", 0, "", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			file, fn := applicationSource(runtime.Frame{Function: tt.function, File: tt.filename, Line: tt.line})
			if file != tt.wantFile || fn != tt.wantFunc {
				t.Fatalf("got %q %q, want %q %q", file, fn, tt.wantFile, tt.wantFunc)
			}
			t.Log(file, fn)
		})
	}
}

func TestIsLoggingFrame(t *testing.T) {
	for _, fn := range []string{loggingPackage + "Error", loggingPackage + "(*LoggerImpl).Error", zerologPackage + ".(*Event).Msg", zerologPackage + "/log.Error"} {
		if !isLoggingFrame(fn) {
			t.Errorf("did not skip %s", fn)
		}
	}
	for _, fn := range []string{projectPackage + "services/auth.Login", projectPackage + "modules/log_test.TestCaller", projectPackage + "modules/logger.Work", "net/http.HandlerFunc.ServeHTTP"} {
		if isLoggingFrame(fn) {
			t.Errorf("unexpected skip %s", fn)
		}
	}
}
