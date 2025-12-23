import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

// Stress test - find breaking point
export const options = {
  stages: [
    { duration: '1m', target: 5000 },    // Ramp to 5k
    { duration: '2m', target: 10000 },   // Ramp to 10k
    { duration: '2m', target: 15000 },   // Ramp to 15k
    { duration: '2m', target: 20000 },   // Ramp to 20k (likely breaking point)
    { duration: '1m', target: 0 },       // Ramp down
  ],
  thresholds: {
    'http_req_duration': ['p(95)<1000'], // Very relaxed
    'errors': ['rate<0.05'],             // 5% error acceptable in stress test
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const res = http.get(`${BASE_URL}/health`);
  check(res, {
    'status is not 0': (r) => r.status !== 0,
  }) || errorRate.add(1);
  
  sleep(0.05);
}
