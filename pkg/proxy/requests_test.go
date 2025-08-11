package proxy

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func defaultTimeout() time.Duration { return 3 * time.Second }

// Given HTTP options Enable gzip & auto referer and a loaded referer list
// When buildRequest is called
// Then headers should NOT explicitly include Accept-Encoding (transport handles it) and SHOULD include a non-empty Referer
func TestBuildRequest_GivenGzipAutoRef_WhenBuild_ThenHeadersSet(t *testing.T) {
	// prepare a referer list so AutoReferer can populate a non-empty value
	f, err := os.CreateTemp("", "refs-*.txt")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	_, _ = f.WriteString("https://ref.example.com\nhttps://ref2.example.com\n")
	_ = f.Close()

	autoRefClient := &Client{}
	if err := autoRefClient.LoadReferers(f.Name()); err != nil {
		t.Fatalf("LoadReferers: %v", err)
	}

	req, err := autoRefClient.buildRequest(RequestParams{
		URL:    "https://example.com/a",
		Method: "GET",
		HTTP: HTTPOptions{
			AcceptGzip:  true,
			AutoReferer: true,
			Headers:     map[string]string{"X-Test": "1"},
		},
	})
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	if req.Header.Get("X-Test") != "1" {
		t.Fatalf("missing header X-Test")
	}
	if req.Header.Get("Referer") == "" {
		t.Fatalf("AutoReferer expected to set Referer to non-empty value")
	}
	// With the new compression strategy, we do not set Accept-Encoding here; Transport will add it.
	if ae := req.Header.Get("Accept-Encoding"); ae != "" {
		t.Fatalf("Accept-Encoding should not be explicitly set on the Request; got %q", ae)
	}
}

// Given gzip encoded response
// When executeRequest is called
// Then body should be transparently decompressed
func TestExecuteRequest_GivenGzip_WhenDo_ThenDecompressed(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, _ = zw.Write([]byte("hello"))
		_ = zw.Close()
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(200)
		_, _ = w.Write(buf.Bytes())
	}))
	defer s.Close()

	c := &Client{}
	httpClient := s.Client()
	req, _ := http.NewRequest("GET", s.URL, nil)

	resp, err := c.executeRequest(httpClient, req, "")
	if err != nil {
		t.Fatalf("executeRequest: %v", err)
	}
	if resp.Status != 200 || string(resp.Body) != "hello" {
		t.Fatalf("got status=%d body=%q", resp.Status, string(resp.Body))
	}
}

// Given followRedirects=false
// When server returns 302
// Then we should receive 302 without following
func TestExecuteRequest_GivenNoFollow_When302_ThenReturns302(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "OK")
	}))
	defer final.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer redirector.Close()

	c := &Client{}
	cli, err := c.getClient("", defaultTimeout(), false, false, false) // followRedirects=false
	if err != nil {
		t.Fatalf("getClient: %v", err)
	}
	req, _ := http.NewRequest("GET", redirector.URL, nil)
	resp, err := c.executeRequest(cli, req, "")
	if err != nil {
		t.Fatalf("executeRequest: %v", err)
	}
	if resp.Status != http.StatusFound {
		t.Fatalf("status=%d want %d", resp.Status, http.StatusFound)
	}
}

// Given followRedirects=true
// When server returns 302
// Then it follows and returns 200
func TestExecuteRequest_GivenFollow_When302_Then200OK(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "OK")
	}))
	defer final.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer redirector.Close()

	c := &Client{}
	cli, err := c.getClient("", defaultTimeout(), false, false, true) // followRedirects=true
	if err != nil {
		t.Fatalf("getClient: %v", err)
	}
	req, _ := http.NewRequest("GET", redirector.URL, nil)
	resp, err := c.executeRequest(cli, req, "")
	if err != nil {
		t.Fatalf("executeRequest: %v", err)
	}
	if resp.Status != 200 || string(resp.Body) != "OK" {
		t.Fatalf("status=%d body=%q", resp.Status, string(resp.Body))
	}
}

// Given random referer and auto referer enabled with random path disabled
// When buildRequest is called
// Then the Referer header should equal the request URL (deterministic)
func TestBuildRequest_GivenRandomEmptyAndAuto_WhenBuild_ThenRefererIsURL(t *testing.T) {
	c := &Client{}
	req, err := c.buildRequest(RequestParams{
		URL:    "https://example.com/test",
		Method: "GET",
		HTTP: HTTPOptions{
			RandomReferer:       true,
			AutoReferer:         true,
			RandomPath:          false, // disable random path to keep Referer == URL deterministic
			RandomPathWithQuery: false,
		},
	})
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	if req.Header.Get("Referer") != "https://example.com/test" {
		t.Fatalf("expected Referer to equal URL, got %q", req.Header.Get("Referer"))
	}
}
