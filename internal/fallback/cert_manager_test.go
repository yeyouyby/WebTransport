package fallback

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"matrix-gateway/internal/sharding"
)

func TestCertManagerGetExactCert(t *testing.T) {
	m := NewCertManager()
	cert := tls.Certificate{}
	m.SetExact("s1.api.com", cert)

	if _, err := m.GetExactCert("s1.api.com"); err != nil {
		t.Fatalf("get exact cert: %v", err)
	}

	if _, err := m.GetExactCert("s2.api.com"); !errors.Is(err, ErrNoExactCert) {
		t.Fatalf("want ErrNoExactCert, got %v", err)
	}
}

func TestParseRangeHeader(t *testing.T) {
	offset, length, ok := parseRangeHeader("bytes=100-199")
	if !ok {
		t.Fatal("expected range to parse")
	}
	if offset != 100 || length != 100 {
		t.Fatalf("unexpected result offset=%d length=%d", offset, length)
	}
}

func TestHandleFallbackWithHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/fallback", nil)
	req.Header.Set("Range", "bytes=0-3")
	w := httptest.NewRecorder()

	handleFallback(w, req, func(_ context.Context, _ uint64, _ uint32, writer io.Writer) error {
		_, err := writer.Write([]byte("test"))
		return err
	})

	if w.Code != http.StatusPartialContent {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if w.Body.String() != "test" {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestHandleFallbackWithHandlerErrorBeforeWrite(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/fallback", nil)
	req.Header.Set("Range", "bytes=0-3")
	w := httptest.NewRecorder()

	handleFallback(w, req, func(_ context.Context, _ uint64, _ uint32, _ io.Writer) error {
		return errors.New("upstream failed")
	})

	if w.Code != http.StatusBadGateway {
		t.Fatalf("unexpected status: %d", w.Code)
	}
}

func TestHandleFallbackWithHandlerErrorAfterPartialWrite(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/fallback", nil)
	req.Header.Set("Range", "bytes=0-3")
	w := httptest.NewRecorder()

	handleFallback(w, req, func(_ context.Context, _ uint64, _ uint32, writer io.Writer) error {
		if _, err := writer.Write([]byte("test")); err != nil {
			return err
		}
		return errors.New("upstream failed after write")
	})

	if w.Code != http.StatusPartialContent {
		t.Fatalf("unexpected status: %d, body: %q", w.Code, w.Body.String())
	}
	if body := w.Body.String(); body != "test" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestHandleShardConfig(t *testing.T) {
	manager, err := sharding.NewManager(sharding.Config{
		ImageHosts:    []string{"s1.api.com", "s2.api.com"},
		VideoHosts:    []string{"s1.api.com", "s2.api.com", "s3.api.com"},
		AudioHosts:    []string{"s1.api.com", "s4.api.com"},
		DefaultScheme: "https",
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/shards", nil)
	w := httptest.NewRecorder()
	handleShardConfig(w, req, manager)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "imageShards") || !strings.Contains(body, "videoShards") || !strings.Contains(body, "audioShards") {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestHandleShardClientScript(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/shard-client.js", nil)
	w := httptest.NewRecorder()
	handleShardClientScript(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "MatrixShardClient") {
		t.Fatalf("script not served correctly")
	}
}

func TestParseRangeHeaderOverflow(t *testing.T) {
	_, _, ok := parseRangeHeader("bytes=0-18446744073709551615")
	if ok {
		t.Fatal("expected overflow range to fail")
	}
}
