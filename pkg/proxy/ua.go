package proxy

import (
	"math/rand"
	"time"
)

// getUASlice returns the current user-agent list snapshot; nil if unset.
// It reads from atomic.Value so it's lock-free for readers.
func (c *Client) getUASlice() []string {
	v := c.uaListVal.Load()
	if v == nil {
		return nil
	}
	if s, ok := v.([]string); ok {
		return s
	}
	return nil
}

// LoadUserAgents loads user agents from a file into an atomic snapshot.
// It ignores empty lines and lines starting with '#'. If the file path is empty,
// it clears the UA list. It also compares mtime to avoid unnecessary reloads.
func (c *Client) LoadUserAgents(path string) error {
	if path == "" {
		// clear list
		c.uaListVal.Store([]string{})
		c.uaListPath = ""
		c.uaListMTime = time.Time{}
		return nil
	}

	lines, mtime, err := readLines(path)
	if err != nil {
		return err
	}

	// If same file and not modified, skip reload
	if c.uaListPath == path && !mtime.After(c.uaListMTime) {
		return nil
	}

	// store snapshot atomically (can be empty)
	c.uaListVal.Store(lines)
	c.uaListPath = path
	c.uaListMTime = mtime

	return nil
}

// getRandomUserAgent returns a UA using lock-free round-robin over the current list.
// It returns an empty string if the list is empty or unset.
func (c *Client) getRandomUserAgent() string {
	list := c.getUASlice()
	if len(list) == 0 {
		return ""
	}

	if c.uaRand == nil {
		c.uaRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	idx := c.uaRand.Intn(len(list))
	return list[idx]
}

// GetRandomUserAgent is the public wrapper to get a random UA string for k6 scripts.
func (c *Client) GetRandomUserAgent() string {
	return c.getRandomUserAgent()
}
