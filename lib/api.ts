import type { SSHKey } from "./api-types";

export const API_TOKEN_KEY = "open-git-auth-token";

export type OrgProfile = {
  id: number;
  login: string;
  name: string;
  type?: string;
  description?: string;
};

export type ApiFieldError = {
  field?: string;
  code?: string;
  message?: string;
  resource?: string;
};

export class ApiError extends Error {
  status: number;
  errors?: ApiFieldError[];

  constructor(status: number, message: string, errors?: ApiFieldError[]) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.errors = errors;
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
      let errors: ApiFieldError[] | undefined;
      try {
        const errorBody = (await response.json()) as {
          message?: string;
          errors?: ApiFieldError[];
        };
        message = errorBody.message ?? message;
        errors = errorBody.errors;
      } catch {
        // ignore JSON parse errors
      }

      if (response.status === 401) {
        this.setToken(null);
        this.router?.push("/login");
      }

      throw new ApiError(response.status, message, errors);
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

  put<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("PUT", path, body);
  }

  del(path: string): Promise<void> {
    return this.request<void>("DELETE", path);
  }

  orgs = {
    create: (data: { login: string; name: string; description?: string }) =>
      this.post<OrgProfile>("/api/v3/orgs", data),
    update: (login: string, data: { name?: string; description?: string }) =>
      this.patch<OrgProfile>(`/api/v3/orgs/${login}`, data),
    delete: (login: string) => this.del(`/api/v3/orgs/${login}`),
    inviteMember: (org: string, username: string, role: string) =>
      this.put<void>(`/api/v3/orgs/${org}/memberships/${username}`, { role }),
    removeMember: (org: string, username: string) =>
      this.del(`/api/v3/orgs/${org}/members/${username}`),
  };

  sshKeys = {
    list: () => this.get<SSHKey[]>("/api/v3/user/keys"),
    create: (title: string, key: string) =>
      this.post<SSHKey>("/api/v3/user/keys", { title, key }),
    remove: (id: string) => this.del("/api/v3/user/keys/" + id),
  };
}
