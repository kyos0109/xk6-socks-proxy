package proxy

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func (c *Client) LoadProxyList(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat proxy list: %w", err)
	}
	c.proxyListLock.RLock()
	samePath := (c.proxyListPath == path)
	sameTime := (samePath && !fi.ModTime().After(c.proxyListMTime))
	c.proxyListLock.RUnlock()
	if sameTime {
		return nil // no change
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read proxy list: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	var proxies []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			proxies = append(proxies, line)
		}
	}
	if len(proxies) == 0 {
		return fmt.Errorf("proxy list is empty")
	}
	c.proxyListLock.Lock()
	c.proxyList = proxies
	c.proxyIndex = 0
	c.proxyListPath = path
	c.proxyListMTime = fi.ModTime()
	c.proxyListLock.Unlock()
	return nil
}

func (c *Client) GetNextProxy() string {
	c.proxyListLock.RLock()
	n := len(c.proxyList)
	c.proxyListLock.RUnlock()
	if n == 0 {
		return ""
	}
	now := time.Now()
	c.proxyListLock.Lock()
	defer c.proxyListLock.Unlock()
	start := c.proxyIndex
	for i := 0; i < len(c.proxyList); i++ {
		idx := (start + i) % len(c.proxyList)
		p := c.proxyList[idx]
		if t, bad := c.badProxies.Load(p); bad {
			if expireAt, ok := t.(time.Time); ok && now.Before(expireAt) {
				continue
			}
			c.badProxies.Delete(p)
		}
		c.proxyIndex = (idx + 1) % len(c.proxyList)
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
