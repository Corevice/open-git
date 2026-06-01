export const API_TOKEN_KEY = "open-git-auth-token";

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

type HttpMethod = "GET" | "POST" | "PATCH" | "DELETE";

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

  del(path: string): Promise<void> {
    return this.request<void>("DELETE", path);
  }
}
