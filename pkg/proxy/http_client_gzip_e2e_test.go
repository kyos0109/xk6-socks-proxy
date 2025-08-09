package proxy

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// local gzip server returning "hello" with Content-Encoding: gzip
func newGzipServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, _ = zw.Write([]byte("hello"))
		_ = zw.Close()

		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}))
}

// Given SkipDecompress=false
// When executing request against a gzip server
// Then body should be transparently decompressed to "hello"
func TestExecuteRequestWithOpts_GivenSkipDecompressFalse_WhenGzipServer_ThenBodyDecompressed(t *testing.T) {
	t.Parallel()
	ts := newGzipServer()
	defer ts.Close()

	c := &Client{}

	// Build a plain request without setting Accept-Encoding.
	// net/http Transport will add gzip and auto-decompress when DisableCompression=false.
	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("new request error: %v", err)
	}

	cli, err := c.getClientWithOpts(
		"", 2*time.Second, false, false, true, false, // skipDecompress=false
	)
	if err != nil {
		t.Fatalf("getClientWithOpts error: %v", err)
	}

	httpOpts := HTTPOptions{
		// AcceptGzip 可由 Transport 自動處理，不需手動設 header
		SkipDecompress: false,
	}

	resp, err := c.executeRequestWithOpts(cli, req, "", httpOpts)
	if err != nil {
		t.Fatalf("executeRequestWithOpts error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.Status)
	}
	if string(resp.Body) != "hello" {
		t.Fatalf("expected decompressed body \"hello\", got %q", string(resp.Body))
	}
}

// Given SkipDecompress=true
// When executing request against a gzip server
// Then body should remain compressed (not equal to "hello"), and likely start with gzip magic bytes
func TestExecuteRequestWithOpts_GivenSkipDecompressTrue_WhenGzipServer_ThenBodyCompressed(t *testing.T) {
	t.Parallel()
	ts := newGzipServer()
	defer ts.Close()

	c := &Client{}

	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("new request error: %v", err)
	}

	cli, err := c.getClientWithOpts(
		"", 2*time.Second, false, false, true, true, // skipDecompress=true
	)
	if err != nil {
		t.Fatalf("getClientWithOpts error: %v", err)
	}

	httpOpts := HTTPOptions{
		SkipDecompress: true,
	}

	resp, err := c.executeRequestWithOpts(cli, req, "", httpOpts)
	if err != nil {
		t.Fatalf("executeRequestWithOpts error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.Status)
	}
	if string(resp.Body) == "hello" {
		t.Fatalf("body should NOT be auto-decompressed when SkipDecompress=true")
	}
	// Optional: gzip magic header verification (0x1f 0x8b)
	if len(resp.Body) >= 2 && !(resp.Body[0] == 0x1f && resp.Body[1] == 0x8b) {
		t.Fatalf("expected gzip-compressed bytes (header 0x1f 0x8b); got first two bytes: %#x %#x", resp.Body[0], resp.Body[1])
	}
}
