package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

type clientKey struct {
	Proxy              string
	Timeout            time.Duration
	Insecure           bool
	DisableH2          bool
	FollowRedirects    bool
	DisableCompression bool
}

func (k clientKey) String() string {
	return fmt.Sprintf("%s|%s|ik:%t|h2off:%t|redir:%t|nocomp:%t",
		k.Proxy, k.Timeout.String(), k.Insecure, k.DisableH2, k.FollowRedirects, k.DisableCompression)
}

func (c *Client) getClientWithOpts(proxyURL string, timeout time.Duration, insecure, disableH2, followRedirects, skipDecompress bool) (*http.Client, error) {
	key := clientKey{
		Proxy:              proxyURL,
		Timeout:            timeout,
		Insecure:           insecure,
		DisableH2:          disableH2,
		FollowRedirects:    followRedirects,
		DisableCompression: skipDecompress,
	}

	if v, ok := c.clients.Load(key.String()); ok {
		return v.(*http.Client), nil
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	tr := &http.Transport{
		DialContext:         dialer.DialContext,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: insecure},
		DisableKeepAlives:   false,
		ForceAttemptHTTP2:   !disableH2,
		MaxIdleConns:        4096,
		MaxIdleConnsPerHost: 1024,
		IdleConnTimeout:     90 * time.Second,
		// SkipDecompress=true => Transport.DisableCompression=true (do not auto-decompress)
		// SkipDecompress=false => Transport.DisableCompression=false (allow auto-decompress)
		DisableCompression: skipDecompress,
	}

	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		switch strings.ToLower(u.Scheme) {
		case "socks5", "socks5h", "socks4":
			auth := proxy.Auth{User: u.User.Username()}
			if pwd, ok := u.User.Password(); ok {
				auth.Password = pwd
			}
			d, err := proxy.SOCKS5("tcp", u.Host, &auth, proxy.Direct)
			if err != nil {
				return nil, err
			}
			type contextDialer interface {
				DialContext(ctx context.Context, network, addr string) (net.Conn, error)
			}
			if cd, ok := d.(contextDialer); ok {
				tr.DialContext = cd.DialContext
			} else {
				tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
					return d.Dial(network, addr)
				}
			}
		default:
			tr.Proxy = http.ProxyURL(u)
		}
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !followRedirects {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	c.clients.Store(key.String(), client)
	return client, nil
}

func (c *Client) getClient(proxyURL string, timeout time.Duration, insecure, disableH2, followRedirects bool) (*http.Client, error) {
	return c.getClientWithOpts(proxyURL, timeout, insecure, disableH2, followRedirects, false)
}
