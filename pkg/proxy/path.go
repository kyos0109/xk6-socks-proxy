package proxy

import (
	"math/rand"
	"net/url"
	"strings"
	"time"
)

// GetRandomPath generates a random path with random extension (no query string).
func (c *Client) GetRandomPath() string {
	return c.randomPath(false)
}

// GetRandomPathWithQuery generates a random path with random extension and random query string.
func (c *Client) GetRandomPathWithQuery() string {
	return c.randomPath(true)
}

// randomPath generates 1~3 segments path with random extension, optionally adding query string.
func (c *Client) randomPath(withQuery bool) string {
	rnd := c.randPath()
	segments := rnd.Intn(3) + 1 // 1~3 segments
	var parts []string
	for i := 0; i < segments; i++ {
		parts = append(parts, randomString(rnd, rnd.Intn(8)+3)) // 3~10 chars per segment
	}

	exts := []string{"html", "php", "asp", "aspx", "jsp", "json", "txt", "png", "jpg", "pdf", "gif", "ico"}
	path := "/" + strings.Join(parts, "/") + "." + exts[rnd.Intn(len(exts))]

	if withQuery {
		qv := url.Values{}
		paramCount := rnd.Intn(3) + 1 // 1~3 params
		for i := 0; i < paramCount; i++ {
			key := randomString(rnd, rnd.Intn(5)+3)
			val := randomString(rnd, rnd.Intn(6)+3)
			qv.Set(key, val)
		}
		path += "?" + qv.Encode()
	}

	return path
}

func (c *Client) randPath() *rand.Rand {
	if c.pathRand == nil {
		c.pathRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return c.pathRand
}

// randomString generates a random lowercase string of given length.
func randomString(rnd *rand.Rand, length int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		b[i] = letters[rnd.Intn(len(letters))]
	}
	return string(b)
}
