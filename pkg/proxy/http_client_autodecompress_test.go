package proxy

import (
	"net/http"
	"testing"
	"time"
)

// Given SkipDecompress=false
// When getClientWithOpts is called
// Then Transport.DisableCompression should be false (auto-decompress enabled by net/http)
func TestGetClientWithOpts_GivenSkipDecompressFalse_WhenBuild_ThenAutoDecompressEnabled(t *testing.T) {
	t.Parallel()
	c := &Client{}

	cli, err := c.getClientWithOpts(
		"",            // no proxy
		2*time.Second, // timeout
		false,         // insecure
		false,         // disableH2
		true,          // followRedirects
		false,         // skipDecompress = false
	)
	if err != nil {
		t.Fatalf("getClientWithOpts error: %v", err)
	}

	tr, ok := cli.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport is not *http.Transport: %T", cli.Transport)
	}
	if tr.DisableCompression {
		t.Fatalf("DisableCompression should be false when SkipDecompress=false (auto-decompress expected)")
	}
}
