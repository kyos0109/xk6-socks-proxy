package proxy

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
)

// LoadUserAgents loads UA list from file (one per line) with mtime caching
func (c *Client) LoadUserAgents(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat ua list: %w", err)
	}
	c.uaLock.RLock()
	samePath := (c.uaListPath == path)
	sameTime := (samePath && !fi.ModTime().After(c.uaListMTime))
	c.uaLock.RUnlock()
	if sameTime {
		return nil // no change
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read ua list: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	var agents []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			agents = append(agents, line)
		}
	}
	if len(agents) == 0 {
		return fmt.Errorf("ua list is empty")
	}

	c.uaLock.Lock()
	c.userAgents = agents
	c.uaListPath = path
	c.uaListMTime = fi.ModTime()
	c.uaLock.Unlock()
	return nil
}

var uaSeq uint64

func (c *Client) getRandomUserAgent() string {
	c.uaLock.RLock()
	n := len(c.userAgents)
	c.uaLock.RUnlock()
	if n == 0 {
		return ""
	}
	idx := int(atomic.AddUint64(&uaSeq, 1)-1) % n
	c.uaLock.RLock()
	ua := c.userAgents[idx]
	c.uaLock.RUnlock()
	return ua
}
