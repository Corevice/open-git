import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL, AUTH_TOKEN, READ_VUS, READ_DURATION } from './config.js';

export const options = {
  vus: READ_VUS,
  duration: READ_DURATION,
  thresholds: {
    'http_req_duration': ['p(95)<300'],
  },
};

const headers = {
  Authorization: `Bearer ${AUTH_TOKEN}`,
  'Content-Type': 'application/json',
};

const queries = [
  {
    name: 'viewer',
    query: '{ viewer { login } }',
  },
  {
    name: 'repository',
    query: '{ repository(owner: "testorg", name: "testrepo") { name } }',
  },
  {
    name: 'issues',
    query:
      '{ repository(owner: "testorg", name: "testrepo") { issues(first: 10) { nodes { number title } } } }',
  },
  {
    name: 'pullRequests',
    query:
      '{ repository(owner: "testorg", name: "testrepo") { pullRequests(first: 10) { nodes { number title } } } }',
  },
];

export default function () {
  const entry = queries[__ITER % queries.length];

  const res = http.post(
    `${BASE_URL}/api/graphql`,
    JSON.stringify({ query: entry.query }),
    { headers, tags: { name: `graphql_${entry.name}` } },
  );
  check(res, {
    [`graphql ${entry.name} status 200`]: (r) => r.status === 200,
  });
}
