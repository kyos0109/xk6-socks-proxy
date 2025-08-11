package proxy

import (
	"math/rand"
	"time"
)

// getRefererSlice returns the current referer list snapshot; nil if unset.
func (c *Client) getRefererSlice() []string {
	v := c.refererListVal.Load()
	if v == nil {
		return nil
	}
	if s, ok := v.([]string); ok {
		return s
	}
	return nil
}

// LoadReferers loads referers from a file into an atomic snapshot.
// It ignores empty lines and lines starting with '#'. If the file path is empty,
// it clears the referer list. It also compares mtime to avoid unnecessary reloads.
func (c *Client) LoadReferers(path string) error {
	if path == "" {
		// clear list
		c.refererListVal.Store([]string{})
		c.refererListPath = ""
		c.refererListMTime = time.Time{}
		return nil
	}

	lines, mtime, err := readLines(path)
	if err != nil {
		return err
	}

	// If same file and not modified, skip reload
	if c.refererListPath == path && !mtime.After(c.refererListMTime) {
		return nil
	}

	// store snapshot atomically
	c.refererListVal.Store(lines)
	c.refererListPath = path
	c.refererListMTime = mtime

	return nil
}

// getRandomReferer returns a referer string randomly chosen from the current list.
// Returns empty string if no referers are set.
func (c *Client) getRandomReferer() string {
	list := c.getRefererSlice()
	if len(list) == 0 {
		return ""
	}

	if c.refererRand == nil {
		c.refererRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	idx := c.refererRand.Intn(len(list))
	return list[idx]
}

// GetRandomReferer is the public wrapper for k6 scripts.
func (c *Client) GetRandomReferer() string {
	return c.getRandomReferer()
}
