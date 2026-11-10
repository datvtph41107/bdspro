package web

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

var errHijackProbe = errors.New("hijack attempted")

type hijackProbeWriter struct{ http.ResponseWriter }

func (w hijackProbeWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errHijackProbe
}

func TestResponseWriterPreservesHijacker(t *testing.T) {
	writer := &responseWriter{ResponseWriter: hijackProbeWriter{httptest.NewRecorder()}}
	_, _, err := writer.Hijack()
	if !errors.Is(err, errHijackProbe) {
		t.Fatalf("Hijack delegation error = %v, want sentinel", err)
	}
}

func TestResponseWriterHijackUnsupported(t *testing.T) {
	writer := &responseWriter{ResponseWriter: httptest.NewRecorder()}
	_, _, err := writer.Hijack()
	if !errors.Is(err, http.ErrNotSupported) {
		t.Fatalf("Hijack unsupported error = %v, want ErrNotSupported", err)
	}
}

func TestResponseWriterStatusFollowsHTTPCommit(t *testing.T) {
	recorder := httptest.NewRecorder()
	writer := &responseWriter{ResponseWriter: recorder}
	if writer.WrittenStatus() != 0 {
		t.Fatalf("new writer reported status = %d", writer.WrittenStatus())
	}
	if _, err := writer.Write([]byte("body")); err != nil {
		t.Fatal(err)
	}
	if writer.WrittenStatus() != http.StatusOK || recorder.Code != http.StatusOK {
		t.Fatalf("write status = %d, recorder = %d", writer.WrittenStatus(), recorder.Code)
	}
}
