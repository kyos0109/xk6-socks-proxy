package proxy

import (
	"fmt"
	"sync"
	"time"

	"go.k6.io/k6/js/modules"
)

// HTTPOptions defines HTTP-specific options for requests
type HTTPOptions struct {
	Timeout            string            `json:"timeout"`
	InsecureSkipVerify bool              `json:"insecureSkipVerify"`
	DisableHTTP2       bool              `json:"disableHTTP2"`
	AutoReferer        bool              `json:"autoReferer"`
	FollowRedirects    bool              `json:"followRedirects"`
	AcceptGzip         bool              `json:"acceptGzip"`
	Headers            map[string]string `json:"headers"`
	RandomUserAgent    bool              `json:"randomUserAgent"`
	UserAgentListPath  string            `json:"userAgentListPath"`
}

// ProxyOptions defines proxy-specific options for requests
type ProxyOptions struct {
	URL      string `json:"url"`
	ListPath string `json:"listPath"`
	Disable  bool   `json:"disable"`
}

// ApplyDefaults fills zero-values from a default HTTPOptions
func (o *HTTPOptions) ApplyDefaults(def HTTPOptions) {
	if o.Timeout == "" && def.Timeout != "" {
		o.Timeout = def.Timeout
	}
	if !o.InsecureSkipVerify {
		o.InsecureSkipVerify = def.InsecureSkipVerify
	}
	if !o.DisableHTTP2 {
		o.DisableHTTP2 = def.DisableHTTP2
	}
	if !o.AutoReferer {
		o.AutoReferer = def.AutoReferer
	}
	if !o.FollowRedirects {
		o.FollowRedirects = def.FollowRedirects
	}
	if !o.AcceptGzip {
		o.AcceptGzip = def.AcceptGzip
	}
	if len(o.Headers) == 0 && len(def.Headers) > 0 {
		o.Headers = map[string]string{}
		for k, v := range def.Headers {
			o.Headers[k] = v
		}
	}
	if !o.RandomUserAgent && def.RandomUserAgent {
		o.RandomUserAgent = true
	}
	if o.UserAgentListPath == "" && def.UserAgentListPath != "" {
		o.UserAgentListPath = def.UserAgentListPath
	}
}

// ApplyDefaults fills zero-values from a default ProxyOptions
func (o *ProxyOptions) ApplyDefaults(def ProxyOptions) {
	if o.URL == "" && def.URL != "" {
		o.URL = def.URL
	}
	if o.ListPath == "" && def.ListPath != "" {
		o.ListPath = def.ListPath
	}
	if !o.Disable {
		o.Disable = def.Disable
	}
}

// RequestParams defines the input parameters for each request (with nested HTTP/Proxy options)
type RequestParams struct {
	URL    string       `json:"url"`
	Method string       `json:"method"`
	Body   string       `json:"body"`
	HTTP   HTTPOptions  `json:"http"`
	Proxy  ProxyOptions `json:"proxy"`
}

// Response defines the output returned to JS
type Response struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
	Error  string `json:"error,omitempty"`
}

// Client implements the k6/x/sockshttp module
type Client struct {
	clients        sync.Map // map[string]*http.Client
	badProxies     sync.Map // map[string]time.Time
	proxyList      []string
	proxyIndex     int
	proxyListLock  sync.RWMutex
	proxyListPath  string
	proxyListMTime time.Time
	badProxyTTL    time.Duration
	defaultHTTP    HTTPOptions
	defaultProxy   ProxyOptions

	// user-agent list cache
	userAgents  []string
	uaLock      sync.RWMutex
	uaListPath  string
	uaListMTime time.Time
}

// New returns a new Client instance
func New() modules.Instance {
	return &Client{badProxyTTL: 5 * time.Minute}
}

func (c *Client) parseRequest(raw any) (RequestParams, error) {
	m, err := asMap(raw)
	if err != nil {
		return RequestParams{}, err
	}
	// move flat keys into nested maps to avoid duplicate decode paths
	m = normalizeKeys(m)
	var r RequestParams
	// top-level
	if v, ok := m["url"]; ok {
		if s, ok := asString(v); ok {
			r.URL = s
		}
	}
	if v, ok := m["method"]; ok {
		if s, ok := asString(v); ok {
			r.Method = s
		}
	}
	if v, ok := m["body"]; ok {
		if s, ok := asString(v); ok {
			r.Body = s
		}
	}
	// nested decode (single path)
	if hv, ok := m["http"]; ok {
		if hm, ok := hv.(map[string]any); ok {
			decodeHTTPOptions(hm, &r.HTTP)
		}
	}
	if pv, ok := m["proxy"]; ok {
		if pm, ok := pv.(map[string]any); ok {
			decodeProxyOptions(pm, &r.Proxy)
		}
	}
	return r, nil
}

// Request performs an HTTP request via SOCKS or HTTP proxy
func (c *Client) Request(raw any) (any, error) {
	params, err := c.parseRequest(raw)
	if err != nil {
		return nil, err
	}

	params.HTTP.ApplyDefaults(c.defaultHTTP)
	params.Proxy.ApplyDefaults(c.defaultProxy)

	if params.Proxy.Disable {
		params.Proxy.URL = ""
		params.Proxy.ListPath = ""
	}

	if params.Proxy.URL == "" && params.Proxy.ListPath != "" {
		_ = c.LoadProxyList(params.Proxy.ListPath)
		params.Proxy.URL = c.GetNextProxy()
	}
	// if proxy unhealthy, bail early
	if params.Proxy.URL != "" {
		if t, bad := c.badProxies.Load(params.Proxy.URL); bad {
			if expireAt, ok := t.(time.Time); ok && time.Now().Before(expireAt) {
				return Response{Error: fmt.Sprintf("proxy marked as unhealthy: %s", params.Proxy.URL)}, nil
			}
			c.badProxies.Delete(params.Proxy.URL)
		}
	}

	// Load UA list if configured for this request
	if params.HTTP.RandomUserAgent && params.HTTP.UserAgentListPath != "" {
		_ = c.LoadUserAgents(params.HTTP.UserAgentListPath)
	}

	timeout := 6 * time.Second
	if d, err := time.ParseDuration(params.HTTP.Timeout); err == nil {
		timeout = d
	}

	client, err := c.getClient(params.Proxy.URL, timeout, params.HTTP.InsecureSkipVerify, params.HTTP.DisableHTTP2, params.HTTP.FollowRedirects)
	if err != nil {
		if params.Proxy.URL != "" {
			c.markBadProxy(params.Proxy.URL)
		}
		return Response{Error: err.Error()}, nil
	}

	req, err := c.buildRequest(params)
	if err != nil {
		return Response{Error: err.Error()}, nil
	}

	resp, err := c.executeRequest(client, req, params.Proxy.URL)
	if err != nil {
		if params.Proxy.URL != "" {
			c.markBadProxy(params.Proxy.URL)
		}
		return Response{Error: err.Error()}, nil
	}
	return *resp, nil
}

func (c *Client) Exports() modules.Exports {
	return modules.Exports{
		Named: map[string]any{
			"request":        c.Request,
			"loadProxyList":  c.LoadProxyList,
			"loadUserAgents": c.LoadUserAgents,
			"configure":      c.Configure,
		},
	}
}

// Configure sets default HTTP and Proxy options
func (c *Client) Configure(raw any) (any, error) {
	m, err := asMap(raw)
	if err != nil {
		return nil, err
	}
	if hv, ok := m["http"]; ok {
		if hm, ok := hv.(map[string]any); ok {
			decodeHTTPOptions(hm, &c.defaultHTTP)
		}
	}
	if pv, ok := m["proxy"]; ok {
		if pm, ok := pv.(map[string]any); ok {
			decodeProxyOptions(pm, &c.defaultProxy)
		}
	}
	return map[string]string{"status": "ok"}, nil
}
