package proxy

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Given a UA file with comments/blank lines
// When LoadUserAgents is called
// Then it loads only valid lines and caches mtime
func TestLoadUserAgents_GivenFile_WhenLoad_ThenCacheAndList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ua.txt")
	err := os.WriteFile(path, []byte(`#c
UA-1

UA-2
`), 0o644)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	var c Client
	if err := c.LoadUserAgents(path); err != nil {
		t.Fatalf("LoadUserAgents: %v", err)
	}
	if got := len(c.userAgents); got != 2 {
		t.Fatalf("len=%d want 2", got)
	}
	if c.uaListMTime.IsZero() {
		t.Fatalf("mtime should be set")
	}
}

// Given a loaded UA list
// When getRandomUserAgent is invoked many times
// Then it returns non-empty values from the list
func TestGetRandomUserAgent_GivenLoaded_WhenPick_ThenNonEmpty(t *testing.T) {
	var c Client
	c.userAgents = []string{"A", "B", "C"}
	for i := 0; i < 10; i++ {
		ua := c.getRandomUserAgent()
		if ua == "" {
			t.Fatalf("empty UA")
		}
	}
}

// Given the UA file unchanged
// When LoadUserAgents called again
// Then mtime should not change (no reload)
func TestLoadUserAgents_GivenUnchanged_WhenReload_ThenMtimeUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ua.txt")
	_ = os.WriteFile(path, []byte("UA-1\n"), 0o644)

	var c Client
	_ = c.LoadUserAgents(path)
	mt := c.uaListMTime

	time.Sleep(5 * time.Millisecond)
	_ = c.LoadUserAgents(path)
	if !c.uaListMTime.Equal(mt) {
		t.Fatalf("mtime changed unexpectedly")
	}
}
