import { check, sleep } from 'k6';
import exec from 'k6/execution';
import socks from 'k6/x/xk6-socks-proxy';

const TARGET_URL  = __ENV.URL || 'https://httpbin.org/headers';
const METHOD      = __ENV.METHOD || 'GET';
const BODY        = __ENV.BODY || '';

const USE_PROXY   = (__ENV.USE_PROXY || '0') === '1';
const PROXY_URL   = __ENV.PROXY_URL || '';
const PROXY_LIST  = __ENV.PROXY_LIST || '';
const UA_LIST     = __ENV.UA_LIST || '';
const RAND_UA     = (__ENV.RAND_UA || '1') === '1';

const TIMEOUT     = __ENV.TIMEOUT || '6s';
const INSECURE    = (__ENV.INSECURE || '0') === '1';
const DISABLE_H2  = (__ENV.DISABLE_H2 || '0') === '1';
const AUTOREFF    = (__ENV.AUTO_REF || '1') === '1';
const FOLLOW_RED  = (__ENV.FOLLOW_RED || '1') === '1';
const ACCEPT_GZIP = (__ENV.ACCEPT_GZIP || '1') === '1';

const RATE        = Number(__ENV.RATE || 1);
const DURATION    = __ENV.DURATION || '2s';
const PRE_VUS     = Number(__ENV.PRE_VUS || 2);
const MAX_VUS     = Number(__ENV.MAX_VUS || 5);

export const options = {
  scenarios: {
    smoke: {
      executor: 'constant-arrival-rate',
      rate: RATE,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: PRE_VUS,
      maxVUs: MAX_VUS,
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
      timeout: TIMEOUT,
      insecureSkipVerify: INSECURE,
      disableHTTP2: DISABLE_H2,
      autoReferer: AUTOREFF,
      followRedirects: FOLLOW_RED,
      acceptGzip: ACCEPT_GZIP,
      randomUserAgent: RAND_UA,
      userAgentListPath: UA_LIST,
      headers: {
        'Accept': '*/*',
      },
    },
    proxy: {
      url: PROXY_URL,
      listPath: PROXY_LIST,
      disable: !USE_PROXY,
    },
  });
}

export default function () {
  const params = {
    url: TARGET_URL,
    method: METHOD,
    body: BODY,
    http: {
      // headers: { 'X-Debug': '1' },
    },
    proxy: {
      url: PROXY_URL,
      listPath: PROXY_LIST,
      disable: !USE_PROXY,
    },
  };

  const res = socks.request(params);

  const ok = check(res, {
    'status ok or has body': (r) => (r.status >= 200 && r.status < 400) || (r.body && r.body.length > 0),
  });

  if (!ok && res.error) {
    console.warn(`iter=${exec.scenario.iterationInTest} error=${res.error}`);
  }
  sleep(0.1);
}