package proxy

import (
	"net/http"
	"testing"
	"time"
)

// Given same cache key
// When getClient called twice
// Then the same *http.Client instance is returned (from cache)
func TestGetClient_GivenSameKey_WhenCalledTwice_ThenCached(t *testing.T) {
	c := &Client{}
	a1, err := c.getClient("", 2*time.Second, false, false, true)
	if err != nil {
		t.Fatalf("getClient a1: %v", err)
	}
	a2, err := c.getClient("", 2*time.Second, false, false, true)
	if err != nil {
		t.Fatalf("getClient a2: %v", err)
	}
	if a1 != a2 {
		t.Fatalf("expected cached instance")
	}
}

// Given different followRedirects flag
// When getClient called with different flags
// Then different *http.Client instances are created
func TestGetClient_GivenDifferentRedirect_WhenCalled_ThenDifferentInstance(t *testing.T) {
	c := &Client{}
	a, _ := c.getClient("", 2*time.Second, false, false, true)
	b, _ := c.getClient("", 2*time.Second, false, false, false)
	if a == b {
		t.Fatalf("expected different clients when followRedirects differs")
	}
}

// Given timeout value
// When getClient creates client
// Then Timeout is applied and Transport type is *http.Transport
func TestGetClient_GivenTimeout_WhenCreate_ThenApplied(t *testing.T) {
	c := &Client{}
	cli, err := c.getClient("", 1500*time.Millisecond, true, true, false)
	if err != nil {
		t.Fatalf("getClient: %v", err)
	}
	if cli.Timeout != 1500*time.Millisecond {
		t.Fatalf("timeout mismatch: %v", cli.Timeout)
	}
	if _, ok := cli.Transport.(*http.Transport); !ok {
		t.Fatalf("unexpected transport type: %T", cli.Transport)
	}
}

func TestSkipDecompressFlag(t *testing.T) {
	c := &Client{}
	client, _ := c.getClientWithOpts("", 5*time.Second, false, false, true, true) // skipDecompress=true

	tr := client.Transport.(*http.Transport)
	if !tr.DisableCompression {
		t.Errorf("Expected DisableCompression=true, got %v", tr.DisableCompression)
	}

	client2, _ := c.getClientWithOpts("", 5*time.Second, false, false, true, false) // skipDecompress=false
	tr2 := client2.Transport.(*http.Transport)
	if tr2.DisableCompression {
		t.Errorf("Expected DisableCompression=false, got %v", tr2.DisableCompression)
	}
}
