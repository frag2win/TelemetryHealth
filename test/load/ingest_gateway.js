import http from 'k6/http';
import { check, sleep } from 'k6';

// k6 Load Test Configuration (PRD §10, §11 Success Criteria, Improvement #18)
// Target load: Sustaining up to 500k spans/sec equivalent
export const options = {
  stages: [
    { duration: '2m', target: 100 }, // ramp up to 100 virtual users
    { duration: '5m', target: 500 }, // scale up to 500 VUs (equivalent load)
    { duration: '3m', target: 500 }, // sustain load
    { duration: '2m', target: 0 },   // ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<50', 'p(99)<100'], // 95% of requests under 50ms, 99% under 100ms
    http_req_failed: ['rate<0.001'],             // request failure rate under 0.1%
  },
};

// Simulated OTLP/JSON export payload representing 1000 spans/req
const payload = JSON.stringify({
  resourceSpans: [
    {
      resource: {
        attributes: [
          { key: 'service.name', value: { stringValue: 'checkout-service' } },
          { key: 'env', value: { stringValue: 'production' } }
        ]
      },
      scopeSpans: [
        {
          scope: { name: 'k6-load-generator' },
          spans: Array.from({ length: 50 }, (_, i) => ({
            traceId: '4bf92f3577b34da6a3ce929d0e0e4736',
            spanId: `00f067aa0ba902${i.toString(16).padStart(2, '0')}`,
            name: 'ProcessPayment',
            kind: 2,
            startTimeUnixNano: Date.now() * 1000000,
            endTimeUnixNano: (Date.now() + 50) * 1000000,
            attributes: [
              { key: 'http.status_code', value: { intValue: 200 } },
              { key: 'user_id', value: { stringValue: `user-${Math.floor(Math.random() * 10000)}` } }
            ]
          }))
        }
      ]
    }
  ]
});

export default function () {
  const url = 'http://localhost:8080/api/v1/remediation/apply'; // Ingest gateway / REST endpoints
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer test-token-k6',
    },
  };

  const res = http.post(url, payload, params);

  check(res, {
    'status is 200': (r) => r.status === 200,
    'transaction is successful': (r) => r.json().status === 'success',
  });

  sleep(0.1); // pacing between requests
}
