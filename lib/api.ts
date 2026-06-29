import type {
  AccessTokenMeta,
  CreateTokenResult,
  OrgMember,
  OrgProfile,
  SSHKey,
  User,
} from "./api-types";
import type {
  CreatePullRequestInput,
  CreateReviewInput,
  MergeInput,
  MergeResponse,
  PullRequest,
  PullRequestFile,
  Review,
  ReviewComment,
  UpdatePullRequestInput,
} from "@/types/pull-request";

export type AccessTokenListItem = AccessTokenMeta & {
  revoked_at: string | null;
};

export const API_TOKEN_KEY = "open-git-auth-token";

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

type HttpMethod = "GET" | "POST" | "PATCH" | "PUT" | "DELETE";

type RouterLike = {
  push: (path: string) => void;
};

export type Branch = { name: string };

export type PullRequestListResult = {
  items: PullRequest[];
  links: Record<string, string>;
};

export type CommitListItem = {
  sha: string;
  commit: {
    message: string;
    author: { name: string; date: string };
  };
  author?: { login: string } | null;
};

function repoApiPath(owner: string, repo: string): string {
  return `/api/v3/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}`;
}

function parseLinkHeader(header: string | null): Record<string, string> {
  const links: Record<string, string> = {};
  if (!header) return links;
  for (const part of header.split(",")) {
    const match = part.match(/<([^>]+)>\s*;\s*rel="([^"]+)"/);
    if (match) links[match[2]] = match[1];
  }
  return links;
}

export class ApiClient {
  private token: string | null = null;
  private router: RouterLike | null;

  constructor(
    private baseURL: string,
    router?: RouterLike,
  ) {
    this.router = router ?? null;
    if (typeof window !== "undefined") {
      this.token = localStorage.getItem(API_TOKEN_KEY);
    }
  }

  setToken(t: string | null): void {
    this.token = t;
    if (typeof window !== "undefined") {
      if (t) {
        localStorage.setItem(API_TOKEN_KEY, t);
      } else {
        localStorage.removeItem(API_TOKEN_KEY);
      }
    }
  }

  getToken(): string | null {
    return this.token;
  }

  private async request<T>(
    method: HttpMethod,
    path: string,
    body?: unknown,
  ): Promise<T> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };

    if (this.token) {
      headers.Authorization = `Bearer ${this.token}`;
    }

    const response = await fetch(`${this.baseURL}${path}`, {
      method,
      headers,
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

      if (response.status === 401) {
        this.setToken(null);
        this.router?.push("/login");
      }

      throw new ApiError(response.status, message);
    }

    const contentType = response.headers.get("content-type");
    if (
      response.status === 204 ||
      !contentType?.includes("application/json")
    ) {
      return undefined as T;
    }

    return response.json() as Promise<T>;
  }

  get<T>(path: string): Promise<T> {
    return this.request<T>("GET", path);
  }

  post<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("POST", path, body);
  }

  patch<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("PATCH", path, body);
  }

  private put<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("PUT", path, body);
  }

  private async requestWithResponse<T>(
    method: HttpMethod,
    path: string,
    body?: unknown,
  ): Promise<{ data: T; response: Response }> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };

    if (this.token) {
      headers.Authorization = `Bearer ${this.token}`;
    }

    const response = await fetch(`${this.baseURL}${path}`, {
      method,
      headers,
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

      if (response.status === 401) {
        this.setToken(null);
        this.router?.push("/login");
      }

      throw new ApiError(response.status, message);
    }

    const contentType = response.headers.get("content-type");
    if (
      response.status === 204 ||
      !contentType?.includes("application/json")
    ) {
      return { data: undefined as T, response };
    }

    const data = (await response.json()) as T;
    return { data, response };
  }

  del(path: string): Promise<void> {
    return this.request<void>("DELETE", path);
  }

  users = {
    getCurrent: () => this.get<User>("/api/v3/user"),
    getByLogin: (login: string) => this.get<User>(`/api/v3/users/${login}`),
    updateCurrent: (
      data: Partial<Pick<User, "name" | "email" | "bio" | "avatar_url">>,
    ) => this.patch<User>("/api/v3/user", data),
  };

  orgs = {
    get: (login: string) => this.get<OrgProfile>(`/api/v3/orgs/${login}`),
    listMembers: (org: string) =>
      this.get<OrgMember[]>(`/api/v3/orgs/${org}/members`),
  };

  sshKeys = {
    list: () => this.get<SSHKey[]>("/api/v3/user/keys"),
    create: (title: string, key: string) =>
      this.post<SSHKey>("/api/v3/user/keys", { title, key }),
    remove: (id: string) => this.del("/api/v3/user/keys/" + id),
  };

  tokens = {
    list: () => this.get<AccessTokenListItem[]>("/api/v3/user/tokens"),
    create: (data: { note: string; scopes: string[]; expires_at?: string }) =>
      this.post<CreateTokenResult>("/api/v3/user/tokens", data),
    revoke: (id: number) => this.del(`/api/v3/user/tokens/${id}`),
  };

  branches = {
    list: (owner: string, repo: string, perPage = 100) =>
      this.get<Branch[]>(
        `${repoApiPath(owner, repo)}/branches?per_page=${perPage}`,
      ),
  };

  pullRequests = {
    list: (
      owner: string,
      repo: string,
      state?: string,
      page?: number,
      perPage?: number,
    ) => {
      const params = new URLSearchParams();
      if (state) params.set("state", state);
      if (page !== undefined) params.set("page", String(page));
      if (perPage !== undefined) params.set("per_page", String(perPage));
      const query = params.toString();
      return this.get<PullRequest[]>(
        `${repoApiPath(owner, repo)}/pulls${query ? `?${query}` : ""}`,
      );
    },
    listWithPagination: async (
      owner: string,
      repo: string,
      state?: string,
      page?: number,
      perPage?: number,
    ): Promise<PullRequestListResult> => {
      const params = new URLSearchParams();
      if (state) params.set("state", state);
      if (page !== undefined) params.set("page", String(page));
      if (perPage !== undefined) params.set("per_page", String(perPage));
      const query = params.toString();
      const { data, response } = await this.requestWithResponse<PullRequest[]>(
        "GET",
        `${repoApiPath(owner, repo)}/pulls${query ? `?${query}` : ""}`,
      );
      return {
        items: data ?? [],
        links: parseLinkHeader(response.headers.get("Link")),
      };
    },
    get: (owner: string, repo: string, pullNumber: number) =>
      this.get<PullRequest>(
        `${repoApiPath(owner, repo)}/pulls/${pullNumber}`,
      ),
    create: (owner: string, repo: string, input: CreatePullRequestInput) =>
      this.post<PullRequest>(`${repoApiPath(owner, repo)}/pulls`, input),
    update: (
      owner: string,
      repo: string,
      pullNumber: number,
      patch: UpdatePullRequestInput,
    ) =>
      this.patch<PullRequest>(
        `${repoApiPath(owner, repo)}/pulls/${pullNumber}`,
        patch,
      ),
    merge: (
      owner: string,
      repo: string,
      pullNumber: number,
      input: MergeInput,
    ) =>
      this.post<MergeResponse>(
        `${repoApiPath(owner, repo)}/pulls/${pullNumber}/merge`,
        input,
      ),
    getFiles: (owner: string, repo: string, pullNumber: number) =>
      this.get<PullRequestFile[]>(
        `${repoApiPath(owner, repo)}/pulls/${pullNumber}/files`,
      ),
    listReviews: (owner: string, repo: string, pullNumber: number) =>
      this.get<Review[]>(
        `${repoApiPath(owner, repo)}/pulls/${pullNumber}/reviews`,
      ),
    createReview: (
      owner: string,
      repo: string,
      pullNumber: number,
      input: CreateReviewInput,
    ) =>
      this.post<Review>(
        `${repoApiPath(owner, repo)}/pulls/${pullNumber}/reviews`,
        input,
      ),
    listReviewComments: (owner: string, repo: string, pullNumber: number) =>
      this.get<ReviewComment[]>(
        `${repoApiPath(owner, repo)}/pulls/${pullNumber}/comments`,
      ),
    listCommits: (
      owner: string,
      repo: string,
      sha: string,
      page?: number,
      perPage?: number,
    ) => {
      const params = new URLSearchParams({ sha });
      if (page !== undefined) params.set("page", String(page));
      if (perPage !== undefined) params.set("per_page", String(perPage));
      return this.get<CommitListItem[]>(
        `${repoApiPath(owner, repo)}/commits?${params.toString()}`,
      );
    },
  };
}

function createAuthenticatedClient(): ApiClient {
  const baseURL =
    process.env.NEXT_PUBLIC_API_BASE_URL ??
    process.env.NEXT_PUBLIC_API_URL ??
    "http://localhost:8080";
  const client = new ApiClient(baseURL);
  if (typeof window !== "undefined") {
    const storedToken = localStorage.getItem(API_TOKEN_KEY);
    if (storedToken) {
      client.setToken(storedToken);
    }
  }
  return client;
}

export function listTokens(): Promise<AccessTokenListItem[]> {
  return createAuthenticatedClient().tokens.list();
}

export function createToken(data: {
  note: string;
  scopes: string[];
  expires_at?: string;
}): Promise<CreateTokenResult> {
  return createAuthenticatedClient().tokens.create(data);
}

export function revokeToken(id: number): Promise<void> {
  return createAuthenticatedClient().tokens.revoke(id);
}

export function listBranches(
  owner: string,
  repo: string,
  perPage?: number,
): Promise<Branch[]> {
  return createAuthenticatedClient().branches.list(owner, repo, perPage);
}

export function listPullRequests(
  owner: string,
  repo: string,
  state?: string,
  page?: number,
  perPage?: number,
): Promise<PullRequest[]> {
  return createAuthenticatedClient().pullRequests.list(
    owner,
    repo,
    state,
    page,
    perPage,
  );
}

export function listPullRequestsWithPagination(
  owner: string,
  repo: string,
  state?: string,
  page?: number,
  perPage?: number,
): Promise<PullRequestListResult> {
  return createAuthenticatedClient().pullRequests.listWithPagination(
    owner,
    repo,
    state,
    page,
    perPage,
  );
}

export function getPullRequest(
  owner: string,
  repo: string,
  pullNumber: number,
): Promise<PullRequest> {
  return createAuthenticatedClient().pullRequests.get(owner, repo, pullNumber);
}

export function createPullRequest(
  owner: string,
  repo: string,
  input: CreatePullRequestInput,
): Promise<PullRequest> {
  return createAuthenticatedClient().pullRequests.create(owner, repo, input);
}

export function updatePullRequest(
  owner: string,
  repo: string,
  pullNumber: number,
  patch: UpdatePullRequestInput,
): Promise<PullRequest> {
  return createAuthenticatedClient().pullRequests.update(
    owner,
    repo,
    pullNumber,
    patch,
  );
}

export function mergePullRequest(
  owner: string,
  repo: string,
  pullNumber: number,
  input: MergeInput,
): Promise<MergeResponse> {
  return createAuthenticatedClient().pullRequests.merge(
    owner,
    repo,
    pullNumber,
    input,
  );
}

export function getPullRequestFiles(
  owner: string,
  repo: string,
  pullNumber: number,
): Promise<PullRequestFile[]> {
  return createAuthenticatedClient().pullRequests.getFiles(
    owner,
    repo,
    pullNumber,
  );
}

export function listReviews(
  owner: string,
  repo: string,
  pullNumber: number,
): Promise<Review[]> {
  return createAuthenticatedClient().pullRequests.listReviews(
    owner,
    repo,
    pullNumber,
  );
}

export function createReview(
  owner: string,
  repo: string,
  pullNumber: number,
  input: CreateReviewInput,
): Promise<Review> {
  return createAuthenticatedClient().pullRequests.createReview(
    owner,
    repo,
    pullNumber,
    input,
  );
}

export function listReviewComments(
  owner: string,
  repo: string,
  pullNumber: number,
): Promise<ReviewComment[]> {
  return createAuthenticatedClient().pullRequests.listReviewComments(
    owner,
    repo,
    pullNumber,
  );
}

export function listPullRequestCommits(
  owner: string,
  repo: string,
  sha: string,
  page?: number,
  perPage?: number,
): Promise<CommitListItem[]> {
  return createAuthenticatedClient().pullRequests.listCommits(
    owner,
    repo,
    sha,
    page,
    perPage,
  );
}
