import type {
  AccessTokenMeta,
  CreateTokenResult,
  OAuthApp,
  OAuthAppCreateInput,
  OAuthAppWithSecret,
  OAuthAuthorizationInfo,
  OrgMember,
  OrgProfile,
  SSHKey,
  User,
} from "./api-types";

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

  oauthApps = {
    list: () => this.get<OAuthApp[]>("/api/v3/user/oauth-apps"),
    create: (input: OAuthAppCreateInput) =>
      this.post<OAuthAppWithSecret>("/api/v3/oauth-apps", input),
    get: (id: string) => this.get<OAuthApp>(`/api/v3/oauth-apps/${id}`),
    update: (id: string, body: Partial<OAuthAppCreateInput>) =>
      this.patch<OAuthApp>(`/api/v3/oauth-apps/${id}`, body),
    regenerateSecret: (id: string) =>
      this.post<{ client_secret: string }>(`/api/v3/oauth-apps/${id}/secret`),
    delete: (id: string) => this.del(`/api/v3/oauth-apps/${id}`),
  };

  userAuthorizations = {
    list: () =>
      this.get<OAuthAuthorizationInfo[]>(
        "/api/v3/user/installations/authorizations",
      ),
    revoke: (appId: string) =>
      this.del(`/api/v3/user/authorizations/${appId}`),
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

type ApiUserSummary = { login: string; avatar_url: string };

function getApiBaseUrl(): string {
  return (
    process.env.NEXT_PUBLIC_API_BASE_URL ??
    process.env.NEXT_PUBLIC_API_URL ??
    "http://localhost:8080"
  );
}

async function fetchWithToken<T>(token: string, path: string): Promise<T> {
  const response = await fetch(`${getApiBaseUrl()}${path}`, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
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

export function getOrgs(
  token: string,
): Promise<{ login: string; avatar_url: string }[]> {
  return fetchWithToken<{ login: string; avatar_url: string }[]>(
    token,
    "/api/v3/user/orgs",
  );
}

export function getCurrentUser(token: string): Promise<ApiUserSummary> {
  return fetchWithToken<ApiUserSummary>(token, "/api/v3/user");
}
