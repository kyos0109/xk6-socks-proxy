import socks from 'k6/x/xk6-socks-proxy';
import { Counter, Rate } from 'k6/metrics';

// Custom metrics
export const extRequests = new Counter('ext_requests');  // Total number of requests
export const extErrors   = new Rate('ext_errors');       // Error rate

export const options = {
  vus: 10,
  duration: '30s',
  thresholds: {
    ext_requests: ['count > 0'],        // At least one request
    ext_errors:   ['rate < 0.01'],      // Error rate < 1%
  },
};

export default function () {
  // Call the Go extension to perform a request
  const res = socks.request({
    url: 'http://httpbin.org/get',
    method: 'GET',
    timeout: '5s',
    // Additional options like proxy, headers, etc.
  });

  // Record metrics
  extRequests.add(1);

  if (!res || res.ok !== true) {
    extErrors.add(1);
  }
}