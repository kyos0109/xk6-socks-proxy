package proxy

import (
	"math/rand"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	leadingSlash    = 1
	avgSegWithSlash = 7
	dotCost         = 1
	avgExtLen       = 4
	queryBudget     = 32
)

// pathRandPool provides concurrency-safe random number generators.
var pathRandPool = sync.Pool{
	New: func() any {
		return rand.New(rand.NewSource(time.Now().UnixNano()))
	},
}

// exts holds common file extensions used when generating random paths.
var exts = [...]string{"html", "php", "asp", "aspx", "jsp", "json", "txt", "png", "jpg", "pdf", "gif", "ico"}

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
	rnd := pathRandPool.Get().(*rand.Rand)
	defer pathRandPool.Put(rnd)

	// Pre-build path segments with capacity to reduce allocations.
	segments := rnd.Intn(3) + 1 // 1~3 segments
	parts := make([]string, 0, segments)
	for i := 0; i < segments; i++ {
		parts = append(parts, randomString(rnd, rnd.Intn(8)+3)) // 3~10 chars per segment
	}

	// Build the final path using a strings.Builder to minimize copies.
	var b strings.Builder
	// Rough capacity hint: "/" + segments*(avg 6+1) + "." + ext + possible query ~ 32
	b.Grow(leadingSlash + segments*avgSegWithSlash + dotCost + avgExtLen + queryBudget)

	b.WriteByte('/')
	b.WriteString(strings.Join(parts, "/"))
	b.WriteByte('.')
	b.WriteString(exts[rnd.Intn(len(exts))])

	if withQuery {
		// Append 1–3 query parameters. Keys/values are lowercase ASCII so QueryEscape is cheap.
		paramCount := rnd.Intn(3) + 1
		sep := byte('?')
		for i := 0; i < paramCount; i++ {
			key := randomString(rnd, rnd.Intn(5)+3)
			val := randomString(rnd, rnd.Intn(6)+3)
			b.WriteByte(sep)
			sep = '&'
			b.WriteString(url.QueryEscape(key))
			b.WriteByte('=')
			b.WriteString(url.QueryEscape(val))
		}
	}

	return b.String()
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
