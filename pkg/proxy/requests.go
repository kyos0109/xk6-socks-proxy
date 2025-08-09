package proxy

import (
	"compress/gzip"
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

	var body io.Reader
	if !(method == http.MethodGet || method == http.MethodHead) && params.Body != "" {
		body = strings.NewReader(params.Body)
	}

	req, err := http.NewRequest(method, params.URL, body)
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
		req.Header.Set("Referer", params.URL)
	}

	if params.HTTP.RandomUserAgent && req.Header.Get("User-Agent") == "" {
		if ua := c.getRandomUserAgent(); ua != "" {
			req.Header.Set("User-Agent", ua)
		}
	}

	if _, ok := req.Header["Accept-Encoding"]; !ok && params.HTTP.AcceptGzip {
		req.Header.Set("Accept-Encoding", "gzip")
	}

	return req, nil
}

func (c *Client) executeRequest(client *http.Client, req *http.Request, proxy string) (*Response, error) {
	resp, err := client.Do(req)
	if err != nil {
		c.markBadProxy(proxy)
		return &Response{
			Error: fmt.Sprintf("request error: %v, proxy: %s, url: %s", err, proxy, req.URL.String()),
		}, nil
	}
	defer resp.Body.Close()
	c.unmarkBadProxy(proxy)

	var reader io.ReadCloser
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return &Response{Error: fmt.Sprintf("gzip decode error: %v", err)}, nil
		}
		defer reader.Close()
	} else {
		reader = resp.Body
	}

	body, _ := io.ReadAll(reader)
	return &Response{Status: resp.StatusCode, Body: string(body)}, nil
}
