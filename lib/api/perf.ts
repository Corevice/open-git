import { API_TOKEN_KEY, ApiError } from "../api";
import { env } from "../env";

export type PerfMetrics = {
  p50_ms: number;
  p95_ms: number;
  p99_ms: number;
  throughput_rps: number;
  error_rate: number;
  total_requests: number;
};

export type PerfRegression = {
  vs_baseline: string;
  flagged: boolean;
  delta_pct?: number;
};

export type PerfBenchmarkItem = {
  id: string;
  scenario_name: string;
  environment: string;
  status: string;
  slo_result: string;
  started_at: string;
  finished_at?: string;
  run_at?: string;
  git_sha?: string;
  metrics: PerfMetrics;
  regression?: PerfRegression;
};

export type PerfBenchmarksResponse = {
  items: PerfBenchmarkItem[];
  next_cursor?: string | null;
};

export type PerfSummaryLatest = {
  scenario_name: string;
  slo_result: string;
  p95_ms?: number;
  error_rate?: number;
  run_at?: string;
};

export type PerfSummaryResponse = {
  latest: PerfSummaryLatest[];
  slo_status: {
    overall: string;
    violations: string[];
  };
  grafana_url?: string;
};

export type PerfJobStatus = {
  status: string;
  benchmark_id?: string | null;
};

export type PerfRunBenchmarkResponse = {
  job_id: string;
  status: string;
};

type HttpMethod = "GET" | "POST";

function resolveBaseUrl(): string {
  return env.NEXT_PUBLIC_API_BASE_URL.replace(/\/$/, "");
}

function buildHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    Accept: "application/json",
    "Content-Type": "application/json",
  };

  if (typeof window !== "undefined") {
    const token = localStorage.getItem(API_TOKEN_KEY);
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }
  }

  return headers;
}

async function request<T>(
  method: HttpMethod,
  path: string,
  body?: unknown,
): Promise<T> {
  const response = await fetch(`${resolveBaseUrl()}${path}`, {
    method,
    headers: buildHeaders(),
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    let message = response.statusText;
    try {
      const errorBody = (await response.json()) as { message?: string };
      message = errorBody.message ?? message;
    } catch {
      // ignore JSON parse errors
    }

    throw new ApiError(response.status, message);
  }

  return response.json() as Promise<T>;
}

export function getPerformanceSummary(): Promise<PerfSummaryResponse> {
  return request<PerfSummaryResponse>("GET", "/api/v3/admin/performance/summary");
}

export function getBenchmarks(params?: {
  limit?: number;
  cursor?: string;
}): Promise<PerfBenchmarksResponse> {
  const search = new URLSearchParams();
  if (params?.limit !== undefined) {
    search.set("limit", String(params.limit));
  }
  if (params?.cursor) {
    search.set("cursor", params.cursor);
  }
  const query = search.toString();
  return request<PerfBenchmarksResponse>(
    "GET",
    `/api/v3/admin/performance/benchmarks${query ? `?${query}` : ""}`,
  );
}

export function getJobStatus(jobId: string): Promise<PerfJobStatus> {
  return request<PerfJobStatus>(
    "GET",
    `/api/v3/admin/performance/jobs/${encodeURIComponent(jobId)}`,
  );
}

export function runBenchmark(): Promise<PerfRunBenchmarkResponse> {
  return request<PerfRunBenchmarkResponse>(
    "POST",
    "/api/v3/admin/performance/benchmarks",
  );
}
