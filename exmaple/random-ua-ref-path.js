import proxy from 'k6/x/xk6-socks-proxy';
import { sleep } from 'k6';

// Environment variable flags
const BASE = __ENV.TARGET_BASE || 'https://httpbin.org';
const WITH_QUERY = (__ENV.WITH_QUERY || 'false').toLowerCase() === 'true';
const USE_RANDOM_REFERER = (__ENV.RANDOM_REFERER || 'true').toLowerCase() === 'true';
const AUTO_REFERER_FALLBACK = (__ENV.AUTO_REFERER || 'true').toLowerCase() === 'true';

export const options = {
  vus: Number(__ENV.VUS || 10),
  duration: __ENV.DURATION || '30s',
};

export function setup() {
  // Load UA / Referer lists if provided
  if (__ENV.UA_LIST) {
    proxy.loadUserAgents(__ENV.UA_LIST);
  }
  if (__ENV.REF_LIST) {
    proxy.loadReferers(__ENV.REF_LIST);
  }
}

export default function () {
  // HTTP parameters with new features
  const params = {
    headers: {
      // 'User-Agent': proxy.getRandomUserAgent(), // Uncomment if you want to set manually
    },
    http: {
      // Compression: true => let Transport set gzip and auto-decompress
      AcceptGzip: true,

      // Referer logic
      RandomReferer: USE_RANDOM_REFERER,
      AutoReferer: AUTO_REFERER_FALLBACK,

      // Path logic
      RandomPath: true,
      RandomPathWithQuery: WITH_QUERY,

      // Enable if you want random UA
      RandomUserAgent: true,
    },
  };

  // The base URL is given; random path & optional query will be appended automatically
  const res = proxy.request('GET', BASE, null, params);

  // Example: log response status and length
  // console.log(`status=${res.status} len=${res.body ? res.body.length : 0}`);

  sleep(1);
}