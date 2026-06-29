import http from 'k6/http';
import { check, group } from 'k6';
import { BASE_URL, AUTH_TOKEN, READ_VUS, READ_DURATION } from './config.js';

export const options = {
  vus: READ_VUS,
  duration: READ_DURATION,
  thresholds: {
    'http_req_duration{scenario:rest_read}': ['p(95)<300', 'p(99)<800'],
    'http_req_failed{scenario:rest_read}': ['rate<0.005'],
  },
  tags: {
    scenario: 'rest_read',
  },
};

const headers = {
  Authorization: `token ${AUTH_TOKEN}`,
  Accept: 'application/json',
};

export default function () {
  group('GET /api/v3/repos/testorg/testrepo', () => {
    const res = http.get(`${BASE_URL}/api/v3/repos/testorg/testrepo`, {
      headers,
      tags: { name: 'get_repo', scenario: 'rest_read' },
    });
    check(res, { 'repo status 200': (r) => r.status === 200 });
  });

  group('GET /api/v3/repos/testorg/testrepo/issues', () => {
    const res = http.get(`${BASE_URL}/api/v3/repos/testorg/testrepo/issues`, {
      headers,
      tags: { name: 'get_issues', scenario: 'rest_read' },
    });
    check(res, { 'issues status 200': (r) => r.status === 200 });
  });

  group('GET /api/v3/repos/testorg/testrepo/pulls', () => {
    const res = http.get(`${BASE_URL}/api/v3/repos/testorg/testrepo/pulls`, {
      headers,
      tags: { name: 'get_pulls', scenario: 'rest_read' },
    });
    check(res, { 'pulls status 200': (r) => r.status === 200 });
  });
}
