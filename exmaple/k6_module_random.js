import http from 'k6/http';
import { check, sleep } from 'k6';
import * as sockshttp from 'k6/x/xk6-socks-proxy';

export const options = {
  vus: __ENV.VUS ? parseInt(__ENV.VUS, 10) : 10,
  duration: __ENV.DURATION || '30s',
};

const BASE_URL = __ENV.BASE_URL || 'https://httpbin.org';
const WITH_QUERY = (__ENV.WITH_QUERY || '1') === '1';

export function setup() {
  // Explicitly load UA and Referer lists (optional if module auto-loads)
  sockshttp.loadUserAgents('./user_agents.txt');
  sockshttp.loadReferers('./referer.txt');
}

export default function () {
  const path = WITH_QUERY
    ? sockshttp.getRandomPathWithQuery()
    : sockshttp.getRandomPath();

  const url = `${BASE_URL}${path}`;
  const ua = sockshttp.getRandomUserAgent() || 'k6-native-fallback/1.0';
  const ref = sockshttp.getRandomReferer() || BASE_URL;

  const headers = {
    'User-Agent': ua,
    'Referer': ref,
    'Accept': '*/*',
  };

  const res = http.get(url, { headers, redirects: 'follow' });

  check(res, {
    'status is 2xx/3xx': (r) => r.status >= 200 && r.status < 400,
  });

  sleep(1);
}