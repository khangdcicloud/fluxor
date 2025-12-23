import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Test configuration
export const options = {
  stages: [
    { duration: '1m', target: 1000 },   // Ramp up to 1k users
    { duration: '2m', target: 5000 },   // Ramp up to 5k users
    { duration: '2m', target: 10000 },  // Ramp up to 10k users
    { duration: '3m', target: 10000 },  // Stay at 10k users for 3 minutes
    { duration: '2m', target: 0 },      // Ramp down to 0 users
  ],
  thresholds: {
    'http_req_duration': ['p(95)<200'],  // 95% of requests must complete below 200ms
    'errors': ['rate<0.001'],            // Error rate must be below 0.1%
    'http_req_failed': ['rate<0.001'],   // Failed requests must be below 0.1%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Generate random user data
function generateUser() {
  const id = Math.floor(Math.random() * 10000);
  return {
    name: `User ${id}`,
    email: `user${id}@example.com`,
    age: Math.floor(Math.random() * 60) + 18,
  };
}

// Generate heavy JSON payload
function generateHeavyPayload() {
  const items = [];
  for (let i = 0; i < 100; i++) {
    items.push({
      id: i,
      name: `Item ${i}`,
      description: `This is a description for item ${i}`,
      price: Math.random() * 1000,
      quantity: Math.floor(Math.random() * 100),
      metadata: {
        created: new Date().toISOString(),
        updated: new Date().toISOString(),
        tags: ['tag1', 'tag2', 'tag3'],
      },
    });
  }
  return { items, total: items.length };
}

export default function () {
  // Test scenario weights
  const scenario = Math.random();

  if (scenario < 0.3) {
    // 30% - Health check (lightweight)
    const res = http.get(`${BASE_URL}/health`);
    check(res, {
      'health status is 200': (r) => r.status === 200,
      'health response has status': (r) => JSON.parse(r.body).status === 'healthy',
    }) || errorRate.add(1);

  } else if (scenario < 0.5) {
    // 20% - Ready check (with metrics)
    const res = http.get(`${BASE_URL}/ready`);
    check(res, {
      'ready status is 200 or 503': (r) => r.status === 200 || r.status === 503,
      'ready response has ready field': (r) => JSON.parse(r.body).ready !== undefined,
    }) || errorRate.add(1);

  } else if (scenario < 0.7) {
    // 20% - Get users (simulated API endpoint)
    const res = http.get(`${BASE_URL}/api/users`, {
      headers: {
        'Authorization': 'Bearer fake-token-for-load-test',
      },
    });
    // Expect 401 (no auth) or 200 (if auth works)
    check(res, {
      'users endpoint responds': (r) => r.status === 401 || r.status === 200,
    }) || errorRate.add(1);

  } else if (scenario < 0.9) {
    // 20% - Heavy JSON endpoint (echo with large payload)
    const payload = generateHeavyPayload();
    const res = http.post(
      `${BASE_URL}/api/echo`,
      JSON.stringify(payload),
      {
        headers: { 'Content-Type': 'application/json' },
      }
    );
    check(res, {
      'echo status is 200': (r) => r.status === 200,
      'echo response has data': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.echo !== undefined;
        } catch {
          return false;
        }
      },
    }) || errorRate.add(1);

  } else {
    // 10% - Metrics endpoint
    const res = http.get(`${BASE_URL}/api/metrics`);
    check(res, {
      'metrics endpoint responds': (r) => r.status === 200 || r.status === 401,
    }) || errorRate.add(1);
  }

  // Think time between requests (50-150ms)
  sleep(Math.random() * 0.1 + 0.05);
}

// Setup function (runs once per VU at the start)
export function setup() {
  console.log(`Starting load test against ${BASE_URL}`);
  console.log('Test configuration:');
  console.log('  - Ramp up to 10k concurrent users over 5 minutes');
  console.log('  - Sustain 10k users for 3 minutes');
  console.log('  - Ramp down over 2 minutes');
  console.log('Thresholds:');
  console.log('  - P95 latency < 200ms');
  console.log('  - Error rate < 0.1%');
}

// Teardown function (runs once at the end)
export function teardown(data) {
  console.log('Load test completed');
}
