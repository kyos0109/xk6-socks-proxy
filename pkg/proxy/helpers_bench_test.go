package proxy

import (
	"encoding/json"
	"testing"
)

// Benchmark normalizeKeys with a representative map containing aliases and nested maps.
func BenchmarkNormalizeKeys_Simple(b *testing.B) {
	in := map[string]any{
		"timeout":            "3s",
		"headers":            map[string]any{"K": "V", "X": 1},
		"proxyListPath":      "./proxies.txt",
		"insecureSkipVerify": "true",
		"proxy":              "socks5://host:1080",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = normalizeKeys(in)
	}
}

// Benchmark asMap starting from JSON bytes.
func BenchmarkAsMap_FromBytes(b *testing.B) {
	data := []byte(`{"a":1,"b":"x","c":true,"d":{"n":3}}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := asMap(data); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark asMap starting from RawMessage.
func BenchmarkAsMap_FromRawMessage(b *testing.B) {
	r := json.RawMessage([]byte(`{"k":"v","z":[1,2,3]}`))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := asMap(r); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark decodeHTTPOptions with a representative map.
func BenchmarkDecodeHTTPOptions(b *testing.B) {
	m := map[string]any{
		"timeout":            "4s",
		"insecureSkipVerify": true,
		"disableHTTP2":       false,
		"autoReferer":        true,
		"followRedirects":    true,
		"acceptGzip":         true,
		"headers":            map[string]any{"A": 1, "B": "x"},
		"randomUserAgent":    true,
		"userAgentListPath":  "./ua.txt",
	}
	var dst HTTPOptions
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		decodeHTTPOptions(m, &dst)
	}
}

// Benchmark decodeProxyOptions with a small map.
func BenchmarkDecodeProxyOptions(b *testing.B) {
	m := map[string]any{
		"url":      "socks5://host:1080",
		"listPath": "./p.txt",
		"disable":  false,
	}
	var dst ProxyOptions
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		decodeProxyOptions(m, &dst)
	}
}
