import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL, AUTH_TOKEN, GIT_VUS, GIT_DURATION } from './config.js';

export const options = {
  vus: GIT_VUS,
  duration: GIT_DURATION,
  thresholds: {
    'http_req_duration': ['p(95)<600'],
    'http_req_failed': ['rate<0.005'],
  },
};

const authHeaders = AUTH_TOKEN
  ? { Authorization: `token ${AUTH_TOKEN}` }
  : {};

export default function () {
  const infoRefsUrl = `${BASE_URL}/testorg/testrepo.git/info/refs?service=git-upload-pack`;

  const infoRefsRes = http.get(infoRefsUrl, {
    headers: {
      ...authHeaders,
      Accept: '*/*',
    },
    tags: { name: 'git_info_refs' },
  });
  check(infoRefsRes, {
    'info/refs status 200': (r) => r.status === 200,
  });

  const uploadPackUrl = `${BASE_URL}/testorg/testrepo.git/git-upload-pack`;

  const uploadPackRes = http.post(uploadPackUrl, '0000', {
    headers: {
      ...authHeaders,
      'Content-Type': 'application/x-git-upload-pack-request',
      Accept: 'application/x-git-upload-pack-result',
    },
    tags: { name: 'git_upload_pack' },
  });
  check(uploadPackRes, {
    'upload-pack status 200': (r) => r.status === 200,
  });
}
