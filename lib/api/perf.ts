import { API_TOKEN_KEY, ApiError } from "../api";
import { env } from "../env";
import type {
  PerfBenchmarkItem,
  PerfBenchmarksResponse,
  PerfJobStatusResponse,
  PerfSummaryResponse,
} from "@/types/perf";

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
  return request<PerfSummaryResponse>("GET", "/api/admin/performance/summary");
}

export function getBenchmarks(params: {
  scenario?: string;
  limit?: number;
  cursor?: string;
}): Promise<PerfBenchmarksResponse> {
  const searchParams = new URLSearchParams();
  if (params.scenario !== undefined) {
    searchParams.set("scenario", params.scenario);
  }
  if (params.limit !== undefined) {
    searchParams.set("limit", String(params.limit));
  }
  if (params.cursor !== undefined) {
    searchParams.set("cursor", params.cursor);
  }
  const query = searchParams.toString();
  const path = query
    ? `/api/admin/performance/benchmarks?${query}`
    : "/api/admin/performance/benchmarks";
  return request<PerfBenchmarksResponse>("GET", path);
}

export function getBenchmarkById(id: string): Promise<PerfBenchmarkItem> {
  return request<PerfBenchmarkItem>(
    "GET",
    `/api/admin/performance/benchmarks/${encodeURIComponent(id)}`,
  );
}

export function runBenchmark(): Promise<{ job_id: string; status: string }> {
  return request<{ job_id: string; status: string }>(
    "POST",
    "/api/admin/performance/run",
  );
}

export function getJobStatus(jobId: string): Promise<PerfJobStatusResponse> {
  return request<PerfJobStatusResponse>(
    "GET",
    `/api/admin/performance/jobs/${encodeURIComponent(jobId)}`,
  );
}
