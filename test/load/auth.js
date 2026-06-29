import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL, AUTH_TOKEN, AUTH_VUS, AUTH_DURATION } from './config.js';

export const options = {
  vus: AUTH_VUS,
  duration: AUTH_DURATION,
  thresholds: {
    'http_req_duration': ['p(95)<200'],
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/api/v3/user`, {
    headers: {
      Authorization: `token ${AUTH_TOKEN}`,
      Accept: 'application/json',
    },
    tags: { name: 'auth_user' },
  });
  check(res, { 'user status 200': (r) => r.status === 200 });
}
