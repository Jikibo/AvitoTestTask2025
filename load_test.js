import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '30s', target: 5 },
    { duration: '1m', target: 5 },
    { duration: '30s', target: 10 },
    { duration: '1m', target: 10 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<300'],
    http_req_failed: ['rate<0.001'],
    errors: ['rate<0.001'],
  },
};

const BASE_URL = 'http://localhost:8080';

export function setup() {
  const teamName = `team-${Date.now()}`;
  
  // Create team
  const teamPayload = JSON.stringify({
    team_name: teamName,
    members: [
      { user_id: `${teamName}-u1`, username: 'User1', is_active: true },
      { user_id: `${teamName}-u2`, username: 'User2', is_active: true },
      { user_id: `${teamName}-u3`, username: 'User3', is_active: true },
      { user_id: `${teamName}-u4`, username: 'User4', is_active: true },
    ],
  });

  const res = http.post(`${BASE_URL}/team/add`, teamPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'team created': (r) => r.status === 201,
  });

  return { teamName };
}

export default function (data) {
  const { teamName } = data;
  const prId = `pr-${__VU}-${__ITER}`;

  let res = http.get(`${BASE_URL}/health`);
  check(res, {
    'health check status is 200': (r) => r.status === 200,
  }) || errorRate.add(1);

  sleep(0.1);

  res = http.get(`${BASE_URL}/team/get?team_name=${teamName}`);
  check(res, {
    'get team status is 200': (r) => r.status === 200,
    'team has members': (r) => JSON.parse(r.body).members.length > 0,
  }) || errorRate.add(1);

  sleep(0.1);

  const prPayload = JSON.stringify({
    pull_request_id: prId,
    pull_request_name: `Test PR ${prId}`,
    author_id: `${teamName}-u1`,
  });

  res = http.post(`${BASE_URL}/pullRequest/create`, prPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  const prCreated = check(res, {
    'create PR status is 201': (r) => r.status === 201,
    'PR has reviewers': (r) => JSON.parse(r.body).pr.assigned_reviewers.length > 0,
    'response time < 300ms': (r) => r.timings.duration < 300,
  });

  if (!prCreated) {
    errorRate.add(1);
    return;
  }

  sleep(0.1);

  res = http.get(`${BASE_URL}/users/getReview?user_id=${teamName}-u2`);
  check(res, {
    'get reviews status is 200': (r) => r.status === 200,
  }) || errorRate.add(1);

  sleep(0.1);

  const mergePayload = JSON.stringify({
    pull_request_id: prId,
  });

  res = http.post(`${BASE_URL}/pullRequest/merge`, mergePayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'merge PR status is 200': (r) => r.status === 200,
    'PR is merged': (r) => JSON.parse(r.body).pr.status === 'MERGED',
  }) || errorRate.add(1);

  sleep(0.1);

  res = http.get(`${BASE_URL}/statistics`);
  check(res, {
    'get statistics status is 200': (r) => r.status === 200,
    'statistics has data': (r) => JSON.parse(r.body).total_prs > 0,
  }) || errorRate.add(1);

  sleep(1);
}

export function teardown(data) {
  console.log('Load test completed');
}
