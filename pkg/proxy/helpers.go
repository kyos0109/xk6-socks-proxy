package proxy

import (
	"encoding/json"
	"fmt"
	"strings"
)

// asString attempts to convert v to string
func asString(v any) (string, bool) {
	switch s := v.(type) {
	case string:
		return s, true
	case []byte:
		return string(s), true
	default:
		return fmt.Sprintf("%v", v), true
	}
}

// asBool attempts to convert v to bool, supporting more string variants
func asBool(v any) (bool, bool) {
	switch b := v.(type) {
	case bool:
		return b, true
	case string:
		s := strings.ToLower(strings.TrimSpace(b))
		switch s {
		case "1", "true", "yes", "y", "on":
			return true, true
		case "0", "false", "no", "n", "off":
			return false, true
		default:
			return false, false
		}
	case float64:
		return b != 0, true
	case int:
		return b != 0, true
	default:
		return false, false
	}
}

// toStringMapString attempts to convert v to map[string]string
func toStringMapString(v any) map[string]string {
	m := map[string]string{}
	switch src := v.(type) {
	case map[string]any:
		for k, vv := range src {
			m[k], _ = asString(vv)
		}
	case map[string]string:
		for k, vv := range src {
			m[k] = vv
		}
	}
	return m
}

// coerce any JS payload to map[string]any
func asMap(raw any) (map[string]any, error) {
	switch v := raw.(type) {
	case map[string]any:
		return v, nil
	case json.RawMessage:
		var m map[string]any
		if err := json.Unmarshal(v, &m); err != nil {
			return nil, err
		}
		return m, nil
	case []byte:
		var m map[string]any
		if err := json.Unmarshal(v, &m); err != nil {
			return nil, err
		}
		return m, nil
	default:
		b, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, err
		}
		return m, nil
	}
}

var keyAlias = map[string]string{
	"timeout":            "http.timeout",
	"insecureSkipVerify": "http.insecureSkipVerify",
	"disableHTTP2":       "http.disableHTTP2",
	"autoReferer":        "http.autoReferer",
	"followRedirects":    "http.followRedirects",
	"acceptGzip":         "http.acceptGzip",
	"headers":            "http.headers",
	"proxyListPath":      "proxy.listPath",
}

// normalizeKeys moves legacy top-level keys into nested `http` or `proxy` maps.
func normalizeKeys(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}

	ensureMap := func(parent string) map[string]any {
		mv, ok := out[parent]
		if ok {
			if mm, ok := mv.(map[string]any); ok {
				return mm
			}
		}
		nm := map[string]any{}
		out[parent] = nm
		return nm
	}

	for k, v := range m {
		if alias, ok := keyAlias[k]; ok {
			parts := strings.SplitN(alias, ".", 2)
			if len(parts) == 2 {
				mm := ensureMap(parts[0])
				mm[parts[1]] = v
				delete(out, k)
			}
		}
	}

	// legacy: proxy can be a string => move to proxy.url
	if pv, ok := out["proxy"]; ok {
		if s, ok := pv.(string); ok {
			pm := ensureMap("proxy")
			pm["url"] = s
		}
	}
	return out
}

// decode helpers to fill option structs
func decodeHTTPOptions(m map[string]any, dst *HTTPOptions) {
	if v, ok := m["timeout"]; ok {
		if s, ok := asString(v); ok {
			dst.Timeout = s
		}
	}
	if v, ok := m["insecureSkipVerify"]; ok {
		if b, ok := asBool(v); ok {
			dst.InsecureSkipVerify = b
		}
	}
	if v, ok := m["disableHTTP2"]; ok {
		if b, ok := asBool(v); ok {
			dst.DisableHTTP2 = b
		}
	}
	if v, ok := m["autoReferer"]; ok {
		if b, ok := asBool(v); ok {
			dst.AutoReferer = b
		}
	}
	if v, ok := m["followRedirects"]; ok {
		if b, ok := asBool(v); ok {
			dst.FollowRedirects = b
		}
	}
	if v, ok := m["acceptGzip"]; ok {
		if b, ok := asBool(v); ok {
			dst.AcceptGzip = b
		}
	}
	if v, ok := m["headers"]; ok {
		if dst.Headers == nil {
			dst.Headers = map[string]string{}
		}
		for k, s := range toStringMapString(v) {
			dst.Headers[k] = s
		}
	}
	if v, ok := m["randomUserAgent"]; ok {
		if b, ok := asBool(v); ok {
			dst.RandomUserAgent = b
		}
	}
	if v, ok := m["userAgentListPath"]; ok {
		if s, ok := asString(v); ok {
			dst.UserAgentListPath = s
		}
	}
}

func decodeProxyOptions(m map[string]any, dst *ProxyOptions) {
	if v, ok := m["url"]; ok {
		if s, ok := asString(v); ok {
			dst.URL = s
		}
	}
	if v, ok := m["listPath"]; ok {
		if s, ok := asString(v); ok {
			dst.ListPath = s
		}
	}
	if v, ok := m["disable"]; ok {
		if b, ok := asBool(v); ok {
			dst.Disable = b
		}
	}
}
