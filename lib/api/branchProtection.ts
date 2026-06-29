import { API_TOKEN_KEY, ApiError } from "../api";
import { env } from "../env";

export interface RequiredStatusChecks {
  strict: boolean;
  contexts: string[];
}

export interface RequiredPullRequestReviews {
  dismiss_stale_reviews: boolean;
  require_code_owner_reviews: boolean;
  required_approving_review_count: number;
}

export interface BranchProtectionRule {
  url?: string;
  pattern: string;
  required_status_checks: RequiredStatusChecks | null;
  required_pull_request_reviews: RequiredPullRequestReviews | null;
  enforce_admins: { enabled: boolean };
  allow_force_pushes: { enabled: boolean };
  allow_deletions: { enabled: boolean };
  required_linear_history: { enabled: boolean };
  required_conversation_resolution: { enabled: boolean };
  restrictions: null;
}

export interface UpsertBranchProtectionRequest {
  required_status_checks: RequiredStatusChecks | null;
  enforce_admins: boolean;
  required_pull_request_reviews: RequiredPullRequestReviews | null;
  restrictions: null;
  allow_force_pushes: boolean;
  allow_deletions: boolean;
  required_linear_history: boolean;
  required_conversation_resolution: boolean;
}

type HttpMethod = "GET" | "PUT" | "DELETE";

function resolveBaseUrl(): string {
  return env.NEXT_PUBLIC_API_BASE_URL.replace(/\/$/, "");
}

function buildHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
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

function protectionPath(owner: string, repo: string, branch: string): string {
  return `/api/v3/repos/${owner}/${repo}/branches/${encodeURIComponent(branch)}/protection`;
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

  const contentType = response.headers.get("content-type");
  if (!contentType?.includes("application/json")) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}

export function listBranchProtections(
  owner: string,
  repo: string,
): Promise<BranchProtectionRule[]> {
  return request<BranchProtectionRule[]>(
    "GET",
    `/api/internal/repos/${owner}/${repo}/branch-protections`,
  );
}

export function getBranchProtection(
  owner: string,
  repo: string,
  branch: string,
): Promise<BranchProtectionRule> {
  return request<BranchProtectionRule>("GET", protectionPath(owner, repo, branch));
}

export function upsertBranchProtection(
  owner: string,
  repo: string,
  branch: string,
  data: UpsertBranchProtectionRequest,
): Promise<BranchProtectionRule> {
  return request<BranchProtectionRule>(
    "PUT",
    protectionPath(owner, repo, branch),
    data,
  );
}

export function deleteBranchProtection(
  owner: string,
  repo: string,
  branch: string,
): Promise<void> {
  return request<void>("DELETE", protectionPath(owner, repo, branch));
}
