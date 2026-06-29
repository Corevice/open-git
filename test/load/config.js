export const BASE_URL = __ENV.K6_TARGET_URL || 'http://localhost:3000';
export const AUTH_TOKEN = __ENV.K6_AUTH_TOKEN || '';

export const READ_VUS = 200;
export const READ_DURATION = '60s';

export const WRITE_VUS = 50;
export const WRITE_DURATION = '60s';

export const GIT_VUS = 20;
export const GIT_DURATION = '60s';

export const AUTH_VUS = 100;
export const AUTH_DURATION = '60s';
