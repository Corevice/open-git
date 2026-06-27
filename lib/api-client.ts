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

function buildHeaders(token?: string): HeadersInit {
  const headers: Record<string, string> = {
    Accept: "application/vnd.github+json",
  };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return headers;
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

export const apiClient = createApiClient(env.NEXT_PUBLIC_API_BASE_URL);
