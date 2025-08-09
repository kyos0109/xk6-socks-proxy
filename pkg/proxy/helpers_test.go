package proxy

import (
	"encoding/json"
	"reflect"
	"testing"
)

// --- asString ---

// Given a string
// When asString is called
// Then it should return the same string and ok=true
func TestAsString_GivenString_WhenConvert_ThenOK(t *testing.T) {
	t.Parallel()
	s, ok := asString("abc")
	if !ok || s != "abc" {
		t.Fatalf("asString failed: got=%q ok=%v", s, ok)
	}
}

// Given a byte slice
// When asString is called
// Then it should convert to string and ok=true
func TestAsString_GivenBytes_WhenConvert_ThenOK(t *testing.T) {
	t.Parallel()
	s, ok := asString([]byte("xyz"))
	if !ok || s != "xyz" {
		t.Fatalf("asString bytes failed: got=%q ok=%v", s, ok)
	}
}

// Given a non-string value
// When asString is called
// Then it should use fmt formatting and ok=true
func TestAsString_GivenOther_WhenConvert_ThenFmtSprintf(t *testing.T) {
	t.Parallel()
	s, ok := asString(123)
	if !ok || s != "123" {
		t.Fatalf("asString other failed: got=%q ok=%v", s, ok)
	}
}

// --- asBool ---

// Given native booleans
// When asBool is called
// Then it should return the same boolean
func TestAsBool_GivenBool_WhenConvert_ThenOK(t *testing.T) {
	t.Parallel()
	if v, ok := asBool(true); !ok || !v {
		t.Fatalf("bool true failed")
	}
	if v, ok := asBool(false); !ok || v {
		t.Fatalf("bool false failed")
	}
}

// Given string variants
// When asBool is called
// Then it should interpret common true/false forms
func TestAsBool_GivenStringVariants_WhenConvert_ThenOK(t *testing.T) {
	t.Parallel()
	trues := []string{"1", "true", "yes", "y", "on", " TRUE ", "On"}
	falses := []string{"0", "false", "no", "n", "off", " False "}
	for _, s := range trues {
		if v, ok := asBool(s); !ok || !v {
			t.Fatalf("want true for %q got %v/%v", s, v, ok)
		}
	}
	for _, s := range falses {
		if v, ok := asBool(s); !ok || v {
			t.Fatalf("want false for %q got %v/%v", s, v, ok)
		}
	}
}

// Given an invalid string
// When asBool is called
// Then it should return ok=false
func TestAsBool_GivenInvalidString_WhenConvert_ThenNotOK(t *testing.T) {
	t.Parallel()
	if _, ok := asBool("maybe"); ok {
		t.Fatalf("want ok=false for invalid string")
	}
}

// Given numbers
// When asBool is called
// Then it should treat zero as false and non-zero as true
func TestAsBool_GivenNumbers_WhenConvert_ThenOK(t *testing.T) {
	t.Parallel()
	if v, ok := asBool(float64(0)); !ok || v {
		t.Fatalf("float 0 -> false expected")
	}
	if v, ok := asBool(float64(1)); !ok || !v {
		t.Fatalf("float 1 -> true expected")
	}
	if v, ok := asBool(int(0)); !ok || v {
		t.Fatalf("int 0 -> false expected")
	}
	if v, ok := asBool(int(42)); !ok || !v {
		t.Fatalf("int 42 -> true expected")
	}
}

// --- toStringMapString ---

// Given map[string]any
// When toStringMapString is called
// Then it should stringify values
func TestToStringMapString_GivenMapAny_WhenConvert_ThenOK(t *testing.T) {
	t.Parallel()
	in := map[string]any{"A": 1, "B": true, "C": "x"}
	out := toStringMapString(in)
	want := map[string]string{"A": "1", "B": "true", "C": "x"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("toStringMapString mismatch: got=%v want=%v", out, want)
	}
}

// Given map[string]string
// When toStringMapString is called
// Then it should return the same map
func TestToStringMapString_GivenMapString_WhenConvert_ThenOK(t *testing.T) {
	t.Parallel()
	in := map[string]string{"K": "V"}
	out := toStringMapString(in)
	if !reflect.DeepEqual(out, in) {
		t.Fatalf("toStringMapString passthrough mismatch: got=%v want=%v", out, in)
	}
}

// --- asMap ---

// Given map[string]any
// When asMap is called
// Then it should return the same content
func TestAsMap_GivenMap_WhenCoerce_ThenSame(t *testing.T) {
	t.Parallel()
	m := map[string]any{"x": 1}
	out, err := asMap(m)
	if err != nil {
		t.Fatalf("asMap map failed: err=%v", err)
	}
	// Accept either int (when map is passed through) or float64 (when JSON round-tripped)
	if v, ok := out["x"]; ok {
		switch n := v.(type) {
		case int:
			if n != 1 {
				t.Fatalf("asMap map value mismatch (int): %v", n)
			}
		case float64:
			if n != 1 {
				t.Fatalf("asMap map value mismatch (float64): %v", n)
			}
		default:
			t.Fatalf("asMap map unexpected type: %T (%v)", v, v)
		}
	} else {
		t.Fatalf("asMap map missing key 'x': %v", out)
	}
}

// Given json.RawMessage
// When asMap is called
// Then it should parse successfully
func TestAsMap_GivenRawMessage_WhenCoerce_ThenParsed(t *testing.T) {
	t.Parallel()
	r := json.RawMessage(`{"a":123}`)
	out, err := asMap(r)
	if err != nil || out["a"].(float64) != 123 {
		t.Fatalf("asMap raw failed: out=%v err=%v", out, err)
	}
}

// Given []byte JSON
// When asMap is called
// Then it should parse successfully
func TestAsMap_GivenBytes_WhenCoerce_ThenParsed(t *testing.T) {
	t.Parallel()
	b := []byte(`{"k":"v"}`)
	out, err := asMap(b)
	if err != nil || out["k"].(string) != "v" {
		t.Fatalf("asMap bytes failed: out=%v err=%v", out, err)
	}
}

// Given a struct
// When asMap is called
// Then it should marshal and unmarshal to a map
func TestAsMap_GivenStruct_WhenCoerce_ThenParsed(t *testing.T) {
	t.Parallel()
	type S struct{ A int }
	out, err := asMap(S{A: 7})
	if err != nil || out["A"].(float64) != 7 {
		t.Fatalf("asMap struct failed: out=%v err=%v", out, err)
	}
}

// --- normalizeKeys ---

// Given legacy/alias keys
// When normalizeKeys is called
// Then keys should be moved under http.* and proxy.* and removed from top-level
func TestNormalizeKeys_GivenAliases_WhenNormalize_ThenNestedAndRemoved(t *testing.T) {
	t.Parallel()
	in := map[string]any{
		"timeout":       "3s",
		"headers":       map[string]any{"X": "1"},
		"proxyListPath": "./p.txt",
	}
	out := normalizeKeys(in)

	httpm, _ := out["http"].(map[string]any)
	if httpm == nil || httpm["timeout"].(string) != "3s" {
		t.Fatalf("missing http.timeout")
	}
	h, _ := httpm["headers"].(map[string]any)
	if h == nil || h["X"].(string) != "1" {
		t.Fatalf("missing http.headers.X")
	}

	proxym, _ := out["proxy"].(map[string]any)
	if proxym == nil || proxym["listPath"].(string) != "./p.txt" {
		t.Fatalf("missing proxy.listPath")
	}

	if _, ok := out["timeout"]; ok {
		t.Fatalf("timeout not moved")
	}
	if _, ok := out["headers"]; ok {
		t.Fatalf("headers not moved")
	}
	if _, ok := out["proxyListPath"]; ok {
		t.Fatalf("proxyListPath not moved")
	}
}

// Given legacy proxy as string
// When normalizeKeys is called
// Then proxy should become proxy.url
func TestNormalizeKeys_GivenLegacyProxyString_WhenNormalize_ThenProxyURLMoved(t *testing.T) {
	t.Parallel()
	in := map[string]any{"proxy": "socks5://host:1080"}
	out := normalizeKeys(in)
	pm, _ := out["proxy"].(map[string]any)
	if pm == nil || pm["url"].(string) != "socks5://host:1080" {
		t.Fatalf("legacy proxy string not normalized: %v", out)
	}
}

// --- decode helpers ---

// Given a map with all HTTP options
// When decodeHTTPOptions is called
// Then HTTPOptions should be fully populated
func TestDecodeHTTPOptions_GivenMap_WhenDecode_ThenStructFilled(t *testing.T) {
	t.Parallel()
	m := map[string]any{
		"timeout":            "5s",
		"insecureSkipVerify": true,
		"disableHTTP2":       true,
		"autoReferer":        true,
		"followRedirects":    false,
		"acceptGzip":         true,
		"headers":            map[string]any{"A": 1, "B": "x"},
		"randomUserAgent":    true,
		"userAgentListPath":  "./ua.txt",
	}
	var dst HTTPOptions
	decodeHTTPOptions(m, &dst)
	if dst.Timeout != "5s" || !dst.InsecureSkipVerify || !dst.DisableHTTP2 || !dst.AutoReferer || dst.FollowRedirects || !dst.AcceptGzip || !dst.RandomUserAgent || dst.UserAgentListPath != "./ua.txt" {
		t.Fatalf("HTTPOptions fields not set correctly: %+v", dst)
	}
	if dst.Headers["A"] != "1" || dst.Headers["B"] != "x" {
		t.Fatalf("headers decode mismatch: %+v", dst.Headers)
	}
}

// Given a map with all Proxy options
// When decodeProxyOptions is called
// Then ProxyOptions should be fully populated
func TestDecodeProxyOptions_GivenMap_WhenDecode_ThenStructFilled(t *testing.T) {
	t.Parallel()
	m := map[string]any{
		"url":      "socks5://host:1080",
		"listPath": "./p.txt",
		"disable":  true,
	}
	var dst ProxyOptions
	decodeProxyOptions(m, &dst)
	if dst.URL != "socks5://host:1080" || dst.ListPath != "./p.txt" || !dst.Disable {
		t.Fatalf("ProxyOptions fields not set correctly: %+v", dst)
	}
}
