const AUTH_TOKEN_KEY = "open-git-auth-token";

export interface WorkflowRun {
  id: number;
  run_number: number;
  name: string;
  status: string;
  conclusion: string | null;
  head_branch: string;
  head_sha: string;
  event: string;
  run_started_at: string | null;
  updated_at: string;
  actor: { login: string } | null;
}

export interface WorkflowStep {
  number: number;
  name: string;
  status: string;
  conclusion: string | null;
}

export interface WorkflowJob {
  id: number;
  run_id: number;
  name: string;
  status: string;
  conclusion: string | null;
  started_at: string | null;
  completed_at: string | null;
  steps: WorkflowStep[];
}

export interface Artifact {
  id: number;
  name: string;
  size_in_bytes: number;
  expired: boolean;
  created_at: string;
  expires_at: string | null;
}

export interface WorkflowRunsResponse {
  total_count: number;
  workflow_runs: WorkflowRun[];
}

export interface WorkflowJobsResponse {
  total_count: number;
  jobs: WorkflowJob[];
}

export interface ArtifactsResponse {
  total_count: number;
  artifacts: Artifact[];
}

export interface RunFilterParams {
  status?: string;
  conclusion?: string;
  branch?: string;
  event?: string;
  actor?: string;
  page?: number;
  per_page?: number;
}

function getAuthHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    Accept: "application/vnd.github+json",
  };

  if (typeof globalThis !== "undefined" && "localStorage" in globalThis) {
    const token = globalThis.localStorage.getItem(AUTH_TOKEN_KEY);
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }
  }

  return headers;
}

function runsBasePath(owner: string, repo: string): string {
  return `/api/repos/${owner}/${repo}/actions/runs`;
}

function buildQueryString(params?: RunFilterParams): string {
  if (!params) return "";

  const searchParams = new URLSearchParams();
  if (params.status !== undefined) searchParams.set("status", params.status);
  if (params.conclusion !== undefined)
    searchParams.set("conclusion", params.conclusion);
  if (params.branch !== undefined) searchParams.set("branch", params.branch);
  if (params.event !== undefined) searchParams.set("event", params.event);
  if (params.actor !== undefined) searchParams.set("actor", params.actor);
  if (params.page !== undefined) searchParams.set("page", String(params.page));
  if (params.per_page !== undefined)
    searchParams.set("per_page", String(params.per_page));

  const query = searchParams.toString();
  return query ? `?${query}` : "";
}

async function request<T>(method: string, path: string): Promise<T> {
  const response = await fetch(path, {
    method,
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
  }

  const contentType = response.headers.get("content-type");
  if (!contentType?.includes("application/json")) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}

export async function listRuns(
  owner: string,
  repo: string,
  params?: RunFilterParams,
): Promise<WorkflowRunsResponse> {
  return request<WorkflowRunsResponse>(
    "GET",
    `${runsBasePath(owner, repo)}${buildQueryString(params)}`,
  );
}

export async function getRun(
  owner: string,
  repo: string,
  runId: number,
): Promise<WorkflowRun> {
  return request<WorkflowRun>("GET", `${runsBasePath(owner, repo)}/${runId}`);
}

export async function listJobs(
  owner: string,
  repo: string,
  runId: number,
): Promise<WorkflowJobsResponse> {
  return request<WorkflowJobsResponse>(
    "GET",
    `${runsBasePath(owner, repo)}/${runId}/jobs`,
  );
}

export async function listArtifacts(
  owner: string,
  repo: string,
  runId: number,
): Promise<ArtifactsResponse> {
  return request<ArtifactsResponse>(
    "GET",
    `${runsBasePath(owner, repo)}/${runId}/artifacts`,
  );
}

export async function rerunRun(
  owner: string,
  repo: string,
  runId: number,
): Promise<void> {
  await request<void>("POST", `${runsBasePath(owner, repo)}/${runId}/rerun`);
}

export async function rerunFailedJobs(
  owner: string,
  repo: string,
  runId: number,
): Promise<void> {
  await request<void>(
    "POST",
    `${runsBasePath(owner, repo)}/${runId}/rerun-failed-jobs`,
  );
}

export async function cancelRun(
  owner: string,
  repo: string,
  runId: number,
): Promise<void> {
  await request<void>("POST", `${runsBasePath(owner, repo)}/${runId}/cancel`);
}
