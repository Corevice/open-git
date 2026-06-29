import { API_TOKEN_KEY, ApiError } from "../api";
import { env } from "../env";

export type WorkflowRun = {
  id: number;
  name: string;
  run_number: number;
  status: string;
  conclusion: string | null;
  head_branch: string;
  head_sha: string;
  run_started_at?: string;
  updated_at?: string;
  actor?: { login: string };
  triggering_actor?: { login: string };
  event?: string;
};

export type WorkflowJob = {
  id: number;
  name: string;
  status: string;
  conclusion: string | null;
  started_at?: string;
  completed_at?: string;
};

export type Artifact = {
  id: number;
  name: string;
  size_in_bytes: number;
  expired?: boolean;
  archive_download_url?: string;
  created_at?: string;
  expires_at?: string;
};

export type ListRunsResponse = {
  workflow_runs?: WorkflowRun[];
  total_count?: number;
};

export type ListJobsResponse = {
  jobs?: WorkflowJob[];
  total_count?: number;
};

export type ListArtifactsResponse = {
  artifacts?: Artifact[];
  total_count?: number;
};

export type ListRunsParams = {
  status?: string;
  branch?: string;
  event?: string;
  page?: number;
  per_page?: number;
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

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}

function repoActionsPath(owner: string, repo: string): string {
  return `/api/v3/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/actions`;
}

export function listRuns(
  owner: string,
  repo: string,
  params: ListRunsParams = {},
): Promise<ListRunsResponse> {
  const search = new URLSearchParams();
  if (params.status) search.set("status", params.status);
  if (params.branch) search.set("branch", params.branch);
  if (params.event) search.set("event", params.event);
  if (params.page !== undefined) search.set("page", String(params.page));
  if (params.per_page !== undefined) {
    search.set("per_page", String(params.per_page));
  }
  const query = search.toString();
  return request<ListRunsResponse>(
    "GET",
    `${repoActionsPath(owner, repo)}/runs${query ? `?${query}` : ""}`,
  );
}

export function getRun(
  owner: string,
  repo: string,
  runId: string,
): Promise<WorkflowRun> {
  return request<WorkflowRun>(
    "GET",
    `${repoActionsPath(owner, repo)}/runs/${encodeURIComponent(runId)}`,
  );
}

export function listJobs(
  owner: string,
  repo: string,
  runId: string,
): Promise<ListJobsResponse> {
  return request<ListJobsResponse>(
    "GET",
    `${repoActionsPath(owner, repo)}/runs/${encodeURIComponent(runId)}/jobs`,
  );
}

export function listArtifacts(
  owner: string,
  repo: string,
  runId: string,
): Promise<ListArtifactsResponse> {
  return request<ListArtifactsResponse>(
    "GET",
    `${repoActionsPath(owner, repo)}/runs/${encodeURIComponent(runId)}/artifacts`,
  );
}

export function cancelRun(
  owner: string,
  repo: string,
  runId: string,
): Promise<void> {
  return request<void>(
    "POST",
    `${repoActionsPath(owner, repo)}/runs/${encodeURIComponent(runId)}/cancel`,
  );
}

export function rerunRun(
  owner: string,
  repo: string,
  runId: string,
): Promise<void> {
  return request<void>(
    "POST",
    `${repoActionsPath(owner, repo)}/runs/${encodeURIComponent(runId)}/rerun`,
  );
}
