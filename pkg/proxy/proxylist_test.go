package proxy

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Given a proxies file
// When LoadProxyList is called
// Then it loads valid entries
func TestLoadProxyList_GivenFile_WhenLoad_ThenList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proxies.txt")
	_ = os.WriteFile(path, []byte(`#c
socks5://p1:1080

http://p2:8080
socks5h://p3:1080
`), 0o644)

	c := &Client{}
	if err := c.LoadProxyList(path); err != nil {
		t.Fatalf("LoadProxyList: %v", err)
	}
	if got := len(c.proxyList); got != 3 {
		t.Fatalf("len=%d want 3", got)
	}
}

// Given 3 proxies loaded
// When GetNextProxy called 4 times
// Then it wraps to the first
func TestGetNextProxy_GivenRotation_WhenWrap_ThenFirstAgain(t *testing.T) {
	c := &Client{proxyList: []string{"p1", "p2", "p3"}}
	p1 := c.GetNextProxy()
	p2 := c.GetNextProxy()
	p3 := c.GetNextProxy()
	p4 := c.GetNextProxy()
	if p1 == "" || p2 == "" || p3 == "" || p4 == "" {
		t.Fatalf("empty proxy in rotation")
	}
	if p4 != p1 {
		t.Fatalf("wrap expected %q got %q", p1, p4)
	}
}

// Given a bad proxy quarantine TTL
// When markBadProxy called
// Then GetNextProxy skips it during TTL
func TestBadProxyCache_GivenTTL_WhenMarkedBad_ThenSkipped(t *testing.T) {
	c := &Client{
		proxyList:   []string{"a", "b", "c"},
		badProxyTTL: 150 * time.Millisecond,
	}
	c.markBadProxy("b")
	seen := map[string]bool{}
	for i := 0; i < 6; i++ {
		seen[c.GetNextProxy()] = true
	}
	if seen["b"] {
		t.Fatalf("bad proxy should be skipped while in TTL")
	}
}

// Given a bad proxy that was quarantined
// When TTL elapses
// Then it is retried again
func TestBadProxyCache_GivenTTLExpired_WhenNext_ThenRetried(t *testing.T) {
	c := &Client{
		proxyList:   []string{"a", "b", "c"},
		badProxyTTL: 50 * time.Millisecond,
	}
	c.markBadProxy("b")
	time.Sleep(c.badProxyTTL + 10*time.Millisecond)

	seen := map[string]bool{}
	for i := 0; i < 6; i++ {
		seen[c.GetNextProxy()] = true
	}
	if !seen["b"] {
		t.Fatalf("expected b to be retried after TTL")
	}
}
