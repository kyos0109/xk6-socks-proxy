package proxy

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// LoadProxyList atomically replaces the in-memory proxy list snapshot.
// It ignores empty lines and lines starting with '#'. When the file path is empty,
// it clears the list. It compares mtime to avoid unnecessary reloads.
func (c *Client) LoadProxyList(path string) error {
	if path == "" {
		// clear list
		c.proxyListVal.Store([]string{})
		c.proxyListPath = ""
		c.proxyListMTime = time.Time{}
		return nil
	}

	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat proxy list: %w", err)
	}
	// If same file and not modified, skip reload
	if c.proxyListPath == path && !fi.ModTime().After(c.proxyListMTime) {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read proxy list: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	proxies := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		proxies = append(proxies, line)
	}
	// Store snapshot atomically (can be empty)
	c.proxyListVal.Store(proxies)
	c.proxyListPath = path
	c.proxyListMTime = fi.ModTime()
	// NOTE: we intentionally do not reset the round-robin cursor (proxyRR)
	// to avoid concentrating traffic on index 0 right after reload.
	return nil
}

// GetNextProxy returns the next healthy proxy using lock-free round-robin over the current snapshot.
// If the list is empty or all proxies are currently marked as bad (not yet expired), it returns an empty string.
func (c *Client) GetNextProxy() string {
	v := c.proxyListVal.Load()
	if v == nil {
		return ""
	}
	list, ok := v.([]string)
	if !ok || len(list) == 0 {
		return ""
	}

	n := len(list)
	if n == 1 {
		p := list[0]
		// quick bad-proxy check
		if t, bad := c.badProxies.Load(p); bad {
			if expireAt, ok := t.(time.Time); ok && time.Now().Before(expireAt) {
				return ""
			}
			c.badProxies.Delete(p)
		}
		return p
	}

	start := int(c.proxyRR.Add(1)-1) % n
	now := time.Now()
	for i := 0; i < n; i++ {
		idx := (start + i) % n
		p := list[idx]
		if t, bad := c.badProxies.Load(p); bad {
			if expireAt, ok := t.(time.Time); ok && now.Before(expireAt) {
				continue
			}
			c.badProxies.Delete(p)
		}
		return p
	}
	return ""
}

func (c *Client) markBadProxy(p string) {
	if p == "" {
		return
	}
	c.badProxies.Store(p, time.Now().Add(c.badProxyTTL))
}

func (c *Client) unmarkBadProxy(p string) {
	if p == "" {
		return
	}
	c.badProxies.Delete(p)
}
