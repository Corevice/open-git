#!/usr/bin/env bash
set -e

if [ "$#" -ne 3 ]; then
  echo "Usage: submit-results.sh <scenario_name> <environment> <k6_json_summary>" >&2
  exit 1
fi

SCENARIO_NAME="$1"
ENVIRONMENT="$2"
K6_JSON_SUMMARY="$3"

if [ -z "${BENCHMARK_API_URL}" ]; then
  echo "BENCHMARK_API_URL is required" >&2
  exit 1
fi

if [ -z "${INTERNAL_PERF_TOKEN}" ]; then
  echo "INTERNAL_PERF_TOKEN is required" >&2
  exit 1
fi

if [ ! -f "${K6_JSON_SUMMARY}" ]; then
  echo "k6 JSON summary file not found: ${K6_JSON_SUMMARY}" >&2
  exit 1
fi

P50=$(jq -r '.metrics."http_req_duration".values."p(50)" // 0' "${K6_JSON_SUMMARY}")
P95=$(jq -r '.metrics."http_req_duration".values."p(95)" // 0' "${K6_JSON_SUMMARY}")
P99=$(jq -r '.metrics."http_req_duration".values."p(99)" // 0' "${K6_JSON_SUMMARY}")
THROUGHPUT=$(jq -r '.metrics.http_reqs.values.rate // 0' "${K6_JSON_SUMMARY}")
ERROR_RATE=$(jq -r '.metrics.http_req_failed.values.rate // 0' "${K6_JSON_SUMMARY}")
TOTAL_REQUESTS=$(jq -r '.metrics.http_reqs.values.count // 0' "${K6_JSON_SUMMARY}")

STARTED_AT=$(jq -r '.state.testRunDurationMs // 0' "${K6_JSON_SUMMARY}")
FINISHED_AT=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
STARTED_AT_ISO=$(date -u -d "@$(($(date +%s) - STARTED_AT / 1000))" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ")

GIT_SHA="${GIT_SHA:-}"

PAYLOAD=$(jq -n \
  --arg scenario_name "${SCENARIO_NAME}" \
  --arg environment "${ENVIRONMENT}" \
  --arg started_at "${STARTED_AT_ISO}" \
  --arg finished_at "${FINISHED_AT}" \
  --arg git_sha "${GIT_SHA}" \
  --argjson p50 "${P50}" \
  --argjson p95 "${P95}" \
  --argjson p99 "${P99}" \
  --argjson throughput "${THROUGHPUT}" \
  --argjson error_rate "${ERROR_RATE}" \
  --argjson total_requests "${TOTAL_REQUESTS}" \
  '{
    scenario_name: $scenario_name,
    environment: $environment,
    started_at: $started_at,
    finished_at: $finished_at,
    git_sha: (if $git_sha == "" then null else $git_sha end),
    metrics: {
      p50_ms: $p50,
      p95_ms: $p95,
      p99_ms: $p99,
      throughput_rps: $throughput,
      error_rate: $error_rate,
      total_requests: $total_requests
    }
  }')

HTTP_CODE=$(curl -s -o /tmp/submit-results-response.json -w '%{http_code}' \
  -X POST "${BENCHMARK_API_URL}/internal/perf/benchmarks" \
  -H "Content-Type: application/json" \
  -H "X-Internal-Token: ${INTERNAL_PERF_TOKEN}" \
  -d "${PAYLOAD}")

if [ "${HTTP_CODE}" -ne 201 ]; then
  echo "Failed to submit benchmark results: HTTP ${HTTP_CODE}" >&2
  cat /tmp/submit-results-response.json >&2
  exit 1
fi

echo "Benchmark results submitted successfully (HTTP 201)"
