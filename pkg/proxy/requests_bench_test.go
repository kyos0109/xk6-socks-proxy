package proxy

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Benchmark buildRequest with a simple GET and minimal headers.
func BenchmarkBuildRequest_SimpleGET(b *testing.B) {
	c := &Client{}
	params := RequestParams{
		URL:    "https://example.com/path",
		Method: "GET",
		HTTP: HTTPOptions{
			AcceptGzip:  true,
			AutoReferer: true,
			Headers: map[string]string{
				"X-Foo": "bar",
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.buildRequest(params); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark buildRequest with a POST body and multiple headers.
func BenchmarkBuildRequest_PostWithBodyAndHeaders(b *testing.B) {
	c := &Client{}
	c.uaListVal.Store([]string{"UA/1.0"})
	bigHeaders := map[string]string{
		"Content-Type": "application/json",
		"X-H1":         "v1",
		"X-H2":         "v2",
		"X-H3":         "v3",
		"X-H4":         "v4",
		"X-H5":         "v5",
		"X-H6":         "v6",
		"X-H7":         "v7",
		"X-H8":         "v8",
		"X-H9":         "v9",
		"X-H10":        "v10",
	}
	params := RequestParams{
		URL:    "https://example.com/api",
		Method: "POST",
		Body:   `{"a":1,"b":"x"}`,
		HTTP: HTTPOptions{
			AcceptGzip:      true,
			AutoReferer:     true,
			RandomUserAgent: true,
			Headers:         bigHeaders,
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.buildRequest(params); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark executeRequest against a local server returning 200 OK (no gzip).
func BenchmarkExecuteRequest_200OK(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer ts.Close()

	c := &Client{}
	cli := ts.Client()
	req, _ := http.NewRequest(http.MethodGet, ts.URL, nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.executeRequest(cli, req, ""); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark executeRequest when the response is gzip encoded.
func BenchmarkExecuteRequest_Gzip(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, _ = zw.Write([]byte("hello world"))
		_ = zw.Close()
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}))
	defer ts.Close()

	c := &Client{}
	cli := ts.Client()
	req, _ := http.NewRequest(http.MethodGet, ts.URL, nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.executeRequest(cli, req, ""); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark executeRequest with redirect behavior disabled (client returns 302 without following).
func BenchmarkExecuteRequest_NoFollowRedirects302(b *testing.B) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	}))
	defer final.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer redirector.Close()

	c := &Client{}
	cli := &http.Client{
		Timeout: 2 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // do not follow
		},
	}
	req, _ := http.NewRequest(http.MethodGet, redirector.URL, nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.executeRequest(cli, req, ""); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark executeRequest with redirect following enabled (should end with 200 OK).
func BenchmarkExecuteRequest_FollowRedirects200(b *testing.B) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	}))
	defer final.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer redirector.Close()

	c := &Client{}
	cli := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, redirector.URL, nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.executeRequest(cli, req, ""); err != nil {
			b.Fatal(err)
		}
	}
}
