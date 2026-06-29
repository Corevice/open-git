import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL, AUTH_TOKEN, WRITE_VUS, WRITE_DURATION } from './config.js';

export const options = {
  vus: WRITE_VUS,
  duration: WRITE_DURATION,
  thresholds: {
    'http_req_duration': ['p(95)<600'],
  },
};

const headers = {
  Authorization: `token ${AUTH_TOKEN}`,
  Accept: 'application/json',
  'Content-Type': 'application/json',
};

const MAX_TRACKED_ISSUES = 50;
const createdIssueIds = [];

export default function () {
  const title = `load-test-issue-${__VU}-${__ITER}-${Date.now()}`;
  const body = `Synthetic issue created by k6 load test VU=${__VU} iter=${__ITER}`;

  const createRes = http.post(
    `${BASE_URL}/api/v3/repos/testorg/testrepo/issues`,
    JSON.stringify({ title, body }),
    { headers, tags: { name: 'create_issue' } },
  );
  check(createRes, { 'create issue status 201': (r) => r.status === 201 });

  let issueNumber = null;
  try {
    const parsed = createRes.json();
    issueNumber = parsed.number;
  } catch (_) {
    return;
  }

  if (issueNumber !== null) {
    if (createdIssueIds.length >= MAX_TRACKED_ISSUES) {
      createdIssueIds.shift();
    }
    createdIssueIds.push(issueNumber);
  }

  const patchTarget = createdIssueIds[__ITER % createdIssueIds.length];
  if (patchTarget === undefined) {
    return;
  }

  const patchRes = http.patch(
    `${BASE_URL}/api/v3/repos/testorg/testrepo/issues/${patchTarget}`,
    JSON.stringify({ body: `Updated by k6 load test at ${Date.now()}` }),
    { headers, tags: { name: 'patch_issue' } },
  );
  check(patchRes, { 'patch issue status 200': (r) => r.status === 200 });
}
