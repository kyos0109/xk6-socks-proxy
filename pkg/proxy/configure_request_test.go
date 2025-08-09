package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// 1) Configure only: ensure schema-based options do not error
func TestConfigure_Defaults_NoError(t *testing.T) {
	t.Parallel()
	c := &Client{}
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
		t.Fatalf("Configure error: %v", err)
	}
}

// 2) Per-request override only: header should be applied
func TestRequest_PerRequestHeaderOverride(t *testing.T) {
	t.Parallel()
	// Server: returns 200 when X-Req=1, otherwise 400
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
	// Minimal configure to avoid proxy side-effects
	_, err := c.Configure(map[string]any{
		"http":  map[string]any{"timeout": "2s"},
		"proxy": map[string]any{"disable": true},
	})
	if err != nil {
		t.Fatalf("Configure error: %v", err)
	}

	respAny, err := c.Request(map[string]any{
		"url":    ts.URL,
		"method": "GET",
		"http": map[string]any{
			"headers": map[string]any{"X-Req": "1"},
		},
	})
	if err != nil {
		t.Fatalf("Request error: %v", err)
	}

	// Accept both value and pointer response
	var resp Response
	switch v := respAny.(type) {
	case Response:
		resp = v
	case *Response:
		resp = *v
	default:
		t.Fatalf("unexpected response type: %T", respAny)
	}

	if resp.Status != http.StatusOK {
		t.Fatalf("status=%d, want=%d; body=%q err=%q", resp.Status, http.StatusOK, resp.Body, resp.Error)
	}
}

// 3) Client cache only: different timeouts should create different *http.Client
func TestClientCache_DifferentTimeouts(t *testing.T) {
	t.Parallel()
	c := &Client{}

	cli1, err := c.getClient("", 2*time.Second, false, false, true)
	if err != nil {
		t.Fatalf("getClient(2s): %v", err)
	}
	cli2, err := c.getClient("", 3*time.Second, false, false, true)
	if err != nil {
		t.Fatalf("getClient(3s): %v", err)
	}
	if cli1 == cli2 {
		t.Fatalf("expected different *http.Client instances for different timeouts")
	}
}
