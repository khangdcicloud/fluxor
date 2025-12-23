import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

// Spike test - sudden burst of traffic
export const options = {
  stages: [
    { duration: '30s', target: 100 },    // Baseline
    { duration: '10s', target: 5000 },   // Sudden spike
    { duration: '1m', target: 5000 },    // Sustain spike
    { duration: '30s', target: 100 },    // Back to baseline
    { duration: '30s', target: 0 },      // Ramp down
  ],
  thresholds: {
    'http_req_duration': ['p(95)<500'],  // More relaxed for spike
    'errors': ['rate<0.01'],             // 1% error rate acceptable during spike
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const res = http.get(`${BASE_URL}/health`);
  check(res, {
    'status is 200 or 503': (r) => r.status === 200 || r.status === 503,
  }) || errorRate.add(1);
  
  sleep(0.1);
}
