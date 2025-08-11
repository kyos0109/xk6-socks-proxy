package proxy

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"go.k6.io/k6/js/modules"
)

// HTTPOptions defines HTTP-specific options for requests
type HTTPOptions struct {
	Timeout             string            `json:"timeout"`
	InsecureSkipVerify  bool              `json:"insecureSkipVerify"`
	DisableHTTP2        bool              `json:"disableHTTP2"`
	AutoReferer         bool              `json:"autoReferer"`
	FollowRedirects     bool              `json:"followRedirects"`
	AcceptGzip          bool              `json:"acceptGzip"`
	Headers             map[string]string `json:"headers"`
	RandomUserAgent     bool              `json:"randomUserAgent"`
	UserAgentListPath   string            `json:"userAgentListPath"`
	DiscardBody         bool              `json:"discardBody"`
	SkipDecompress      bool              `json:"skipDecompress"`
	RandomPathWithQuery bool              `json:"randomPathWithQuery"`
	RandomPath          bool              `json:"randomPath"`

	// Presence flags (not serialized). True when user explicitly supplied the value in request/script.
	DiscardBodyProvided    bool `json:"-"`
	SkipDecompressProvided bool `json:"-"`
}

// ProxyOptions defines proxy-specific options for requests
type ProxyOptions struct {
	URL      string `json:"url"`
	ListPath string `json:"listPath"`
	Disable  bool   `json:"disable"`
}

// ApplyDefaults fills zero-values from a default HTTPOptions in a predictable way.
// Strings adopt defaults when empty; booleans adopt defaults only when the user did not explicitly provide them.
func (o *HTTPOptions) ApplyDefaults(def HTTPOptions) {
	// Strings
	if o.Timeout == "" && def.Timeout != "" {
		o.Timeout = def.Timeout
	}

	// Booleans without presence tracking: only adopt default when it's true and current is false.
	if !o.InsecureSkipVerify && def.InsecureSkipVerify {
		o.InsecureSkipVerify = true
	}
	if !o.DisableHTTP2 && def.DisableHTTP2 {
		o.DisableHTTP2 = true
	}
	if !o.AutoReferer && def.AutoReferer {
		o.AutoReferer = true
	}
	if !o.FollowRedirects && def.FollowRedirects {
		o.FollowRedirects = true
	}
	if !o.AcceptGzip && def.AcceptGzip {
		o.AcceptGzip = true
	}
	if !o.RandomUserAgent && def.RandomUserAgent {
		o.RandomUserAgent = true
	}

	// Booleans with presence tracking: use defaults only when not explicitly provided by the user.
	if !o.DiscardBodyProvided {
		o.DiscardBody = def.DiscardBody
	}
	if !o.SkipDecompressProvided {
		o.SkipDecompress = def.SkipDecompress
	}

	// Headers: merge defaults without overwriting user-provided keys.
	if len(def.Headers) > 0 {
		if o.Headers == nil {
			o.Headers = map[string]string{}
		}
		for k, v := range def.Headers {
			if _, exists := o.Headers[k]; !exists {
				o.Headers[k] = v
			}
		}
	}

	// Paths
	if o.UserAgentListPath == "" && def.UserAgentListPath != "" {
		o.UserAgentListPath = def.UserAgentListPath
	}

	if !o.RandomUserAgent && def.RandomUserAgent {
		o.RandomUserAgent = false
	}

	if !o.RandomPath && def.RandomPath {
		o.RandomPath = false
	}

	if !o.RandomPathWithQuery && def.RandomPathWithQuery {
		o.RandomPathWithQuery = false
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
	Body   []byte `json:"body"`
	Error  string `json:"error,omitempty"`
}

// Client implements the k6/x/sockshttp module
type Client struct {
	clients        sync.Map     // map[string]*http.Client
	badProxies     sync.Map     // map[string]time.Time
	proxyListVal   atomic.Value // holds []string
	proxyRR        atomic.Uint64
	proxyListPath  string
	proxyListMTime time.Time
	badProxyTTL    time.Duration
	defaultHTTP    HTTPOptions
	defaultProxy   ProxyOptions

	// user-agent list cache (atomic/modern fields)
	uaListVal   atomic.Value // holds []string
	uaRand      *rand.Rand
	uaListPath  string
	uaListMTime time.Time

	// referer
	refererListVal   atomic.Value
	refererListPath  string
	refererListMTime time.Time
	refererRand      *rand.Rand

	// random path
	pathRand *rand.Rand
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
			// Remember presence so explicit false won't be overwritten by defaults
			if _, exists := hm["discardBody"]; exists {
				r.HTTP.DiscardBodyProvided = true
			}
			if _, exists := hm["skipDecompress"]; exists {
				r.HTTP.SkipDecompressProvided = true
			}
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

	client, err := c.getClientWithOpts(
		params.Proxy.URL,
		timeout,
		params.HTTP.InsecureSkipVerify,
		params.HTTP.DisableHTTP2,
		params.HTTP.FollowRedirects,
		params.HTTP.SkipDecompress,
	)
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

	resp, err := c.executeRequestWithOpts(client, req, params.Proxy.URL, params.HTTP)
	if err != nil {
		if params.Proxy.URL != "" {
			c.markBadProxy(params.Proxy.URL)
		}
		return Response{Error: err.Error()}, nil
	}
	return *resp, nil
}

func (c *Client) DefaultConfig() (any, error) {
	return map[string]any{
		"http":  c.defaultHTTP,
		"proxy": c.defaultProxy,
	}, nil
}

func (c *Client) Preview(raw any) (any, error) {
	params, err := c.parseRequest(raw)
	if err != nil {
		return nil, err
	}
	params.HTTP.ApplyDefaults(c.defaultHTTP)
	params.Proxy.ApplyDefaults(c.defaultProxy)
	return params, nil
}

func (c *Client) Exports() modules.Exports {
	return modules.Exports{
		Named: map[string]any{
			"request":                c.Request,
			"loadProxyList":          c.LoadProxyList,
			"loadUserAgents":         c.LoadUserAgents,
			"configure":              c.Configure,
			"defaultConfig":          c.DefaultConfig,
			"preview":                c.Preview,
			"getRandomUserAgent":     c.GetRandomUserAgent,
			"loadReferers":           c.LoadReferers,
			"getRandomReferer":       c.GetRandomReferer,
			"getRandomPath":          c.GetRandomPath,
			"getRandomPathWithQuery": c.GetRandomPathWithQuery,
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
