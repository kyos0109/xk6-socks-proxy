package proxy

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeProxiesFile(t *testing.T, dir string, lines []string) string {
	t.Helper()
	path := filepath.Join(dir, "proxies.txt")
	data := ""
	for _, l := range lines {
		data += l + "\n"
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write proxies: %v", err)
	}
	return path
}

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
	// verify by rotating through the list via public API
	s := map[string]bool{}
	s[c.GetNextProxy()] = true
	s[c.GetNextProxy()] = true
	s[c.GetNextProxy()] = true
	if len(s) != 3 {
		t.Fatalf("unique proxies=%d want 3", len(s))
	}
}

// Given 3 proxies loaded
// When GetNextProxy called 4 times
// Then it wraps to the first
func TestGetNextProxy_GivenRotation_WhenWrap_ThenFirstAgain(t *testing.T) {
	c := &Client{}
	dir := t.TempDir()
	path := writeProxiesFile(t, dir, []string{"p1", "p2", "p3"})
	if err := c.LoadProxyList(path); err != nil {
		t.Fatalf("LoadProxyList: %v", err)
	}

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
	c := &Client{badProxyTTL: 150 * time.Millisecond}
	dir := t.TempDir()
	path := writeProxiesFile(t, dir, []string{"a", "b", "c"})
	if err := c.LoadProxyList(path); err != nil {
		t.Fatalf("LoadProxyList: %v", err)
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
	c := &Client{badProxyTTL: 50 * time.Millisecond}
	dir := t.TempDir()
	path := writeProxiesFile(t, dir, []string{"a", "b", "c"})
	if err := c.LoadProxyList(path); err != nil {
		t.Fatalf("LoadProxyList: %v", err)
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
