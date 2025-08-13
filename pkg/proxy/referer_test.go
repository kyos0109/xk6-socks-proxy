package proxy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReferersAndRandom(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "referers.txt")

	content := "https://example.com\nhttps://google.com\n"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write referer file: %v", err)
	}

	c := &Client{}

	if err := c.LoadReferers(file); err != nil {
		t.Fatalf("LoadReferers failed: %v", err)
	}
	if got := len(c.getRefererSlice()); got != 2 {
		t.Fatalf("expected 2 referers, got %d", got)
	}

	ref := c.getRandomReferer()
	if ref != "https://example.com" && ref != "https://google.com" {
		t.Fatalf("unexpected referer: %s", ref)
	}

	oldTime := c.refererListMTime
	if err := c.LoadReferers(file); err != nil {
		t.Fatalf("LoadReferers failed on reload: %v", err)
	}
	if c.refererListMTime != oldTime {
		t.Fatalf("mtime changed unexpectedly on reload")
	}

	if err := c.LoadReferers(""); err != nil {
		t.Fatalf("LoadReferers failed to clear: %v", err)
	}
	if len(c.getRefererSlice()) != 0 {
		t.Fatalf("expected referer list to be empty after clear")
	}
}

func TestGetRandomRefererEmptyList(t *testing.T) {
	c := &Client{}
	if got := c.getRandomReferer(); got != "" {
		t.Fatalf("expected empty string from empty referer list, got %q", got)
	}
}

func TestLoadReferersIgnoresCommentsAndEmptyLines(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "referers.txt")
	content := "# comment\n\nhttps://example.com\n   \nhttps://google.com\n"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write referer file: %v", err)
	}

	c := &Client{}
	if err := c.LoadReferers(file); err != nil {
		t.Fatalf("LoadReferers failed: %v", err)
	}
	if got := len(c.getRefererSlice()); got != 2 {
		t.Fatalf("expected 2 referers after ignoring comments/empty lines, got %d", got)
	}
}
