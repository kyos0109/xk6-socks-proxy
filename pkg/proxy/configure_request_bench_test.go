package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// BenchmarkConfigure_Defaults measures the performance of Client.Configure
func BenchmarkConfigure_Defaults(b *testing.B) {
	c := &Client{}
	for i := 0; i < b.N; i++ {
		_, err := c.Configure(map[string]any{
			"http": map[string]any{
				"timeout":           "2s",
				"acceptGzip":        true,
				"autoReferer":       true,
				"randomUserAgent":   false,
				"userAgentListPath": "",
				"headers":           map[string]any{"Accept": "*/*", "X-Default": "yes"},
			},
			"proxy": map[string]any{
				"disable": true,
			},
		})
		if err != nil {
			b.Fatalf("Configure error: %v", err)
		}
	}
}

// BenchmarkRequest_PerRequestHeaderOverride measures performance of per-request header overrides
func BenchmarkRequest_PerRequestHeaderOverride(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Req") == "1" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("BAD"))
	}))
	defer ts.Close()

	c := &Client{}
	_, err := c.Configure(map[string]any{
		"http":  map[string]any{"timeout": "2s"},
		"proxy": map[string]any{"disable": true},
	})
	if err != nil {
		b.Fatalf("Configure error: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Request(map[string]any{
			"url":    ts.URL,
			"method": "GET",
			"http": map[string]any{
				"headers": map[string]any{"X-Req": "1"},
			},
		})
		if err != nil {
			b.Fatalf("Request error: %v", err)
		}
	}
}

// BenchmarkClientCache_DifferentTimeouts measures performance of creating clients with different timeouts
func BenchmarkClientCache_DifferentTimeouts(b *testing.B) {
	c := &Client{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.getClient("", 2*time.Second, false, false, true)
		if err != nil {
			b.Fatalf("getClient(2s): %v", err)
		}
		_, err = c.getClient("", 3*time.Second, false, false, true)
		if err != nil {
			b.Fatalf("getClient(3s): %v", err)
		}
	}
}
