import { check, sleep } from 'k6';
import * as sockshttp from 'k6/x/xk6-socks-proxy';

export const options = {
  vus: __ENV.VUS ? parseInt(__ENV.VUS, 10) : 10,
  duration: __ENV.DURATION || '30s',
};

const BASE_URL   = __ENV.BASE_URL || 'https://httpbin.org';
const TIMEOUT    = __ENV.TIMEOUT || '6s';
const WITH_QUERY = (__ENV.WITH_QUERY || '1') === '1';

export default function () {
  const params = {
    url: BASE_URL,
    method: 'GET',
    http: {
      timeout: TIMEOUT,
      randomUserAgent: true,
      randomReferer: true,
      autoReferer: true,         // Fallback to URL if referer list is empty
      followRedirects: true,
      acceptGzip: true,
      randomPath: !WITH_QUERY,
      randomPathWithQuery: WITH_QUERY,
      // Optional: specify custom file paths
      // userAgentListPath: './user_agents.txt',
      // refererListPath: './referer.txt',
    },
    proxy: {
      disable: true, // Set false + configure URL or listPath if proxy needed
    },
  };

  const res = sockshttp.request(params);

  check(res, {
    'status is 2xx/3xx': (r) => r && r.status >= 200 && r.status < 400,
  });

  sleep(1);
}