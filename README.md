# xk6-socks-proxy

This extension adds SOCKS/HTTP proxy support, proxy rotation, unhealthy proxy caching, gzip, auto-Referer, redirect control, HTTP/2 toggle, and random User-Agent selection to k6 via a small JS API.

## Build (with xk6)

```bash
# Install xk6
go install go.k6.io/xk6/cmd/xk6@latest

# Build k6 with this extension
xk6 build --with github.com/kyos0109/xk6-socks-proxy@latest -o k6-socks
```

The command above produces a custom `k6` binary that includes this module.

## What this module exports (JS)

- `configure(opts)` – set default HTTP/Proxy options (used as fallbacks for each request)
- `request(params)` – perform one HTTP request using the configured transport (SOCKS/HTTP proxy, TLS flags, etc.)
- `loadProxyList(path)` – load/refresh a proxy list file (one proxy per line)
- `loadUserAgents(path)` – load/refresh a User‑Agent list file (one UA per line)

> Module import path (JS): `import mod from 'k6/x/xk6-socks-proxy'`

## Options schema

### `configure(opts)`
```jsonc
{
  "http": {
    "timeout": "6s",                // string duration
    "insecureSkipVerify": false,     // skip TLS verify
    "disableHTTP2": false,           // force HTTP/1.1 when true
    "autoReferer": true,             // set Referer to request URL when not provided
    "followRedirects": true,         // follow redirects or return 3xx
    "acceptGzip": true,              // add Accept-Encoding: gzip
    "randomUserAgent": false,        // pick UA from userAgents list when true
    "userAgentListPath": "./user_agents.txt", // file with one UA per line
    "headers": {                     // default headers (merged per request)
      "Accept": "*/*"
    }
  },
  "proxy": {
    "url": "",                      // single proxy URL (e.g. socks5h://user:pass@host:1080)
    "listPath": "./proxies.txt",    // path to proxy list file (one per line)
    "disable": false                 // disable all proxy usage when true
  }
}
```

### `request(params)`
```jsonc
{
  "url": "https://httpbin.org/headers",
  "method": "GET",                  // default GET
  "body": "",                       // request body for non-GET
  "http": {                           // optional per-request overrides
    "headers": { "X-Debug": "1" },
    "randomUserAgent": true,
    "userAgentListPath": "./user_agents.txt",
    "timeout": "10s",
    "insecureSkipVerify": false,
    "disableHTTP2": false,
    "autoReferer": true,
    "followRedirects": true,
    "acceptGzip": true
  },
  "proxy": {                          // optional per-request overrides
    "url": "",                       // single proxy URL
    "listPath": "./proxies.txt",     // proxy list file
    "disable": false
  }
}
```

The response returned to JS is an object:
```jsonc
{ "status": 200, "body": "...", "error": "" }
```

## Proxy list format (`proxies.txt`)

- Supported schemes: `socks4`, `socks4a`, `socks5`, `socks5h`, `http`, `https`
- One proxy per line
- Lines starting with `#` are comments
- Authentication is supported via standard URL userinfo

**Example:**
```
socks5h://username:password@your-proxy-host-1:1080
socks5://your-proxy-host-2:1080
http://your-proxy-host-3:8080
https://your-proxy-host-4:8443
```

> The module maintains a health cache: failed proxies are temporarily avoided and retried later.

## User‑Agent list format (`user_agents.txt`)

- One User‑Agent string per line (no quotes)
- Lines starting with `#` are comments

**Example:**
```
Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36
Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Safari/605.1.15
Mozilla/5.0 (X11; Linux x86_64; rv:115.0) Gecko/20100101 Firefox/115.0
```

## Example k6 script

```javascript
import { check, sleep } from 'k6';
import socks from 'k6/x/xk6-socks-proxy';

export const options = {
  scenarios: {
    smoke: {
      executor: 'constant-arrival-rate',
      rate: 50,
      timeUnit: '1s',
      duration: '1m',
      preAllocatedVUs: 50,
      maxVUs: 200,
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<2000'],
  },
};

export function setup() {
  socks.configure({
    http: {
      timeout: '6s',
      insecureSkipVerify: false,
      disableHTTP2: false,
      autoReferer: true,
      followRedirects: true,
      acceptGzip: true,
      randomUserAgent: true,
      userAgentListPath: './user_agents.txt',
      headers: { 'Accept': '*/*' },
    },
    proxy: {
      url: '',
      listPath: './proxies.txt',
      disable: false,
    },
  });

  // Optional explicit preload (otherwise lazy-loaded on first use)
  socks.loadUserAgents('./user_agents.txt');
  socks.loadProxyList('./proxies.txt');
}

export default function () {
  const res = socks.request({
    url: __ENV.URL || 'https://httpbin.org/headers',
    method: 'GET',
    http: {
      // Per-request overrides are optional, e.g. custom header
      // headers: { 'X-Debug': '1' },
    },
    proxy: {
      // You can pin a single proxy just for this request if needed
      // url: 'socks5h://user:pass@host:1080',
    },
  });

  check(res, {
    'status is OK or has body': (r) => (r.status >= 200 && r.status < 400) || (r.body && r.body.length > 0),
  });

  if (res.error) {
    console.warn(res.error);
  }

  sleep(0.1);
}
```

## Tips
- Place `proxies.txt` and `user_agents.txt` next to your script or provide absolute paths.
- If you set `proxy.disable: true`, no proxy will be used even if `url`/`listPath` are present.
- When `randomUserAgent: true` and no `User-Agent` header is set explicitly, one is chosen from the loaded list.