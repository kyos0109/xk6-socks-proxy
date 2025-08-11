package proxy

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (c *Client) buildRequest(params RequestParams) (*http.Request, error) {
	method := strings.ToUpper(strings.TrimSpace(params.Method))
	if method == "" {
		method = http.MethodGet
	}

	url := params.URL
	if params.HTTP.RandomPath {
		url += c.randomPath(params.HTTP.RandomPathWithQuery)
	}

	var body io.Reader
	if !(method == http.MethodGet || method == http.MethodHead) && params.Body != "" {
		body = strings.NewReader(params.Body)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	hasRef := false
	for k, v := range params.HTTP.Headers {
		if strings.EqualFold(k, "Referer") {
			hasRef = true
		}
		req.Header.Set(k, v)
	}
	if params.HTTP.AutoReferer && !hasRef && params.URL != "" {
		req.Header.Set("Referer", c.getRandomReferer())
	}

	if params.HTTP.RandomUserAgent && req.Header.Get("User-Agent") == "" {
		if ua := c.getRandomUserAgent(); ua != "" {
			req.Header.Set("User-Agent", ua)
		}
	}

	// Compression strategy:
	// - If AcceptGzip is true: do NOT set the header here. Let net/http Transport
	//   add "Accept-Encoding: gzip" automatically and transparently decompress
	//   the response (DisableCompression=false). This avoids manual gzip handling
	//   and reduces allocations.
	// - If AcceptGzip is false: explicitly request identity to avoid compressed
	//   payloads and save CPU on decompression.
	if _, ok := req.Header["Accept-Encoding"]; !ok {
		// Compression strategy:
		// - If AcceptGzip is true: rely on Transport to set gzip and auto-decompress
		// - If AcceptGzip is false: explicitly request identity to avoid decompression
		if !params.HTTP.AcceptGzip {
			req.Header.Set("Accept-Encoding", "identity")
		}
	}

	return req, nil
}

// executeRequestWithOpts performs the HTTP request and uses HTTPOptions to control behavior.
// If httpOpts.DiscardBody is true, it will not read the response body and only return the status code.
func (c *Client) executeRequestWithOpts(client *http.Client, req *http.Request, proxy string, httpOpts HTTPOptions) (*Response, error) {
	resp, err := client.Do(req)
	if err != nil {
		c.markBadProxy(proxy)
		return &Response{
			Error: fmt.Sprintf("request error: %v, proxy: %s, url: %s", err, proxy, req.URL.String()),
		}, nil
	}

	// success path
	c.unmarkBadProxy(proxy)

	if httpOpts.DiscardBody {
		resp.Body.Close()
		return &Response{Status: resp.StatusCode}, nil
	}

	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return &Response{Status: resp.StatusCode, Body: b}, nil
}

func (c *Client) executeRequest(client *http.Client, req *http.Request, proxy string) (*Response, error) {
	return c.executeRequestWithOpts(client, req, proxy, HTTPOptions{})
}
