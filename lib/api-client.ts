import { env } from "./env";

export interface ApiError {
  status: number;
  code: string;
  message: string;
}

export interface ApiClient {
  get<T>(path: string, opts?: { token?: string }): Promise<T>;
  post<T>(
    path: string,
    body: unknown,
    opts?: { token?: string },
  ): Promise<T>;
}

export interface CommitsResult<T> {
  commits: T;
  links: Record<string, string>;
}

export interface MCPVerificationCheck {
  id: string;
  category: "graphql" | "rest" | "auth";
  status: "pass" | "fail" | "skip";
  expected: unknown;
  actual: unknown;
  error: string | null;
  duration_ms: number;
}

export interface MCPVerificationRun {
  run_id: string;
  repository: string;
  overall_status: "compatible" | "partial" | "incompatible" | null;
  executed_at: string;
  status?: "queued" | "running" | "completed" | "errored";
}

export interface MCPJobStatus {
  job_id: string;
  status: "queued" | "running" | "completed" | "errored";
  progress?: number;
}

export interface MCPLatestResult {
  run_id: string;
  repository: string;
  overall_status: "compatible" | "partial" | "incompatible";
  executed_at: string;
  checks: MCPVerificationCheck[];
}

export interface RepoApiClient extends ApiClient {
  getRepo<T>(owner: string, repo: string): Promise<T>;
  getContents<T>(
    owner: string,
    repo: string,
    path?: string,
    ref?: string,
  ): Promise<T>;
  getBranches<T>(owner: string, repo: string): Promise<T>;
  createRef(
    owner: string,
    repo: string,
    ref: string,
    sha: string,
    opts?: { token?: string },
  ): Promise<void>;
  deleteBranch(
    owner: string,
    repo: string,
    branch: string,
    opts?: { token?: string },
  ): Promise<void>;
  updateDefaultBranch(
    owner: string,
    repo: string,
    branch: string,
    opts?: { token?: string },
  ): Promise<void>;
  getCommits<T>(
    owner: string,
    repo: string,
    sha: string,
    page: number,
    perPage?: number,
  ): Promise<CommitsResult<T>>;
  runMCPVerification(
    body: { repository: string; targets?: string[] },
    opts?: { token?: string },
  ): Promise<{ job_id: string; status: string }>;
  getMCPJobStatus(
    jobId: string,
    opts?: { token?: string },
  ): Promise<MCPJobStatus>;
  getMCPLatest(opts?: { token?: string }): Promise<MCPLatestResult>;
  getMCPHistory(opts?: {
    page?: number;
    per_page?: number;
    token?: string;
  }): Promise<MCPVerificationRun[]>;
  deleteMCPRun(runId: string, opts?: { token?: string }): Promise<void>;
}

export function isApiError(err: unknown): err is ApiError {
  return (
    typeof err === "object" &&
    err !== null &&
    "status" in err &&
    typeof (err as ApiError).status === "number" &&
    "message" in err &&
    typeof (err as ApiError).message === "string"
  );
}

export function decodeBase64Content(content: string): string {
  const normalized = content.replace(/\n/g, "");
  return Buffer.from(normalized, "base64").toString("utf-8");
}

export function decodePathSegments(segments: string[]): string[] {
  return segments.map((segment) => decodeURIComponent(segment));
}

export function resolveBranchRef(
  refParam: string | null | undefined,
  branches: { name: string }[],
  defaultBranch: string,
): string {
  const names = new Set(branches.map((branch) => branch.name));
  if (refParam && names.has(refParam)) return refParam;
  if (names.has(defaultBranch)) return defaultBranch;
  return branches[0]?.name ?? defaultBranch ?? "main";
}

export function parseLinkHeader(header: string | null): Record<string, string> {
  const links: Record<string, string> = {};
  if (!header) return links;
  for (const part of header.split(",")) {
    const match = part.match(/<([^>]+)>\s*;\s*rel="([^"]+)"/);
    if (match) links[match[2]] = match[1];
  }
  return links;
}

export function pageFromLinkUrl(url: string): number | null {
  try {
    const page = new URL(url).searchParams.get("page");
    return page ? parseInt(page, 10) : 1;
  } catch {
    return null;
  }
}

function buildHeaders(token?: string): HeadersInit {
  const headers: Record<string, string> = {
    Accept: "application/vnd.github+json",
  };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return headers;
}

function encodeRepoPath(path: string): string {
  if (!path) return "";
  return path.split("/").map(encodeURIComponent).join("/");
}

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let body: { code?: string; message?: string } = {};
    try {
      body = await res.json();
    } catch {
      // ignore non-JSON error bodies
    }
    const error: ApiError = {
      status: res.status,
      code: body.code ?? String(res.status),
      message: body.message ?? res.statusText,
    };
    if (res.status === 401 && typeof window !== "undefined") {
      window.location.href = "/sign-in";
    }
    throw error;
  }
  return res.json() as Promise<T>;
}

export function createApiClient(baseUrl: string): ApiClient {
  const base = baseUrl.replace(/\/$/, "");

  return {
    async get<T>(path: string, opts?: { token?: string }): Promise<T> {
      const res = await fetch(`${base}${path}`, {
        method: "GET",
        headers: buildHeaders(opts?.token),
      });
      return handleResponse<T>(res);
    },

    async post<T>(
      path: string,
      body: unknown,
      opts?: { token?: string },
    ): Promise<T> {
      const res = await fetch(`${base}${path}`, {
        method: "POST",
        headers: {
          ...buildHeaders(opts?.token),
          "Content-Type": "application/json",
        },
        body: JSON.stringify(body),
      });
      return handleResponse<T>(res);
    },
  };
}

export function createRepoApiClient(baseUrl: string): RepoApiClient {
  const base = createApiClient(baseUrl);
  const apiBase = baseUrl.replace(/\/$/, "");

  return {
    ...base,
    getRepo(owner, repo) {
      return base.get(`/repos/${owner}/${repo}`);
    },
    getContents(owner, repo, path = "", ref?) {
      const encodedPath = encodeRepoPath(path);
      const pathSegment = encodedPath ? `/${encodedPath}` : "";
      const query = ref ? `?ref=${encodeURIComponent(ref)}` : "";
      return base.get(`/repos/${owner}/${repo}/contents${pathSegment}${query}`);
    },
    getBranches(owner, repo) {
      return base.get(`/repos/${owner}/${repo}/branches?per_page=100`);
    },
    async createRef(owner, repo, ref, sha, opts) {
      await base.post(
        `/api/v1/repos/${owner}/${repo}/git/refs`,
        { ref, sha },
        opts,
      );
    },
    async deleteBranch(owner, repo, branch, opts) {
      const res = await fetch(
        `${apiBase}/api/v1/repos/${owner}/${repo}/git/refs/heads/${encodeURIComponent(branch)}`,
        {
          method: "DELETE",
          headers: buildHeaders(opts?.token),
        },
      );
      if (!res.ok) {
        let body: { code?: string; message?: string } = {};
        try {
          body = await res.json();
        } catch {
          // ignore non-JSON error bodies
        }
        const error: ApiError = {
          status: res.status,
          code: body.code ?? String(res.status),
          message: body.message ?? res.statusText,
        };
        if (res.status === 401 && typeof window !== "undefined") {
          window.location.href = "/sign-in";
        }
        throw error;
      }
    },
    async updateDefaultBranch(owner, repo, branch, opts) {
      const res = await fetch(`${apiBase}/api/v1/repos/${owner}/${repo}`, {
        method: "PATCH",
        headers: {
          ...buildHeaders(opts?.token),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ default_branch: branch }),
      });
      if (!res.ok) {
        let body: { code?: string; message?: string } = {};
        try {
          body = await res.json();
        } catch {
          // ignore non-JSON error bodies
        }
        const error: ApiError = {
          status: res.status,
          code: body.code ?? String(res.status),
          message: body.message ?? res.statusText,
        };
        if (res.status === 401 && typeof window !== "undefined") {
          window.location.href = "/sign-in";
        }
        throw error;
      }
    },
    async getCommits(owner, repo, sha, page, perPage = 30) {
      const url = `${apiBase}/repos/${owner}/${repo}/commits?sha=${encodeURIComponent(sha)}&per_page=${perPage}&page=${page}`;
      const res = await fetch(url, {
        method: "GET",
        headers: buildHeaders(),
        cache: "no-store",
      });
      if (!res.ok) {
        let body: { code?: string; message?: string } = {};
        try {
          body = await res.json();
        } catch {
          // ignore non-JSON error bodies
        }
        const error: ApiError = {
          status: res.status,
          code: body.code ?? String(res.status),
          message: body.message ?? res.statusText,
        };
        throw error;
      }
      const commits = (await res.json()) as T;
      return {
        commits,
        links: parseLinkHeader(res.headers.get("Link")),
      };
    },
    runMCPVerification(body, opts) {
      return base.post("/api/v1/mcp/verification/run", body, opts);
    },
    getMCPJobStatus(jobId, opts) {
      return base.get(
        `/api/v1/mcp/verification/jobs/${encodeURIComponent(jobId)}`,
        opts,
      );
    },
    getMCPLatest(opts) {
      return base.get("/api/v1/mcp/verification/latest", opts);
    },
    getMCPHistory(opts) {
      const params = new URLSearchParams();
      if (opts?.page != null) params.set("page", String(opts.page));
      if (opts?.per_page != null) params.set("per_page", String(opts.per_page));
      const query = params.toString();
      const path = query
        ? `/api/v1/mcp/verification/history?${query}`
        : "/api/v1/mcp/verification/history";
      return base.get(path, { token: opts?.token });
    },
    async deleteMCPRun(runId, opts) {
      const res = await fetch(
        `${apiBase}/api/v1/mcp/verification/runs/${encodeURIComponent(runId)}`,
        {
          method: "DELETE",
          headers: buildHeaders(opts?.token),
        },
      );
      if (!res.ok) {
        let body: { code?: string; message?: string } = {};
        try {
          body = await res.json();
        } catch {
          // ignore non-JSON error bodies
        }
        const error: ApiError = {
          status: res.status,
          code: body.code ?? String(res.status),
          message: body.message ?? res.statusText,
        };
        if (res.status === 401 && typeof window !== "undefined") {
          window.location.href = "/sign-in";
        }
        throw error;
      }
    },
  };
}

export const apiClient = createRepoApiClient(env.NEXT_PUBLIC_API_BASE_URL);
