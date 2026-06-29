import { API_TOKEN_KEY, ApiError } from "../api";
import { env } from "../env";
import type {
  RegisterRunnerRequest,
  RegistrationTokenResponse,
  Runner,
  RunnerListResponse,
} from "@/types/runner";

type HttpMethod = "GET" | "POST" | "DELETE";

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

function runnersPath(org: string, runnerId?: string): string {
  const base = `/api/v1/${org}/actions/runners`;
  return runnerId !== undefined ? `${base}/${runnerId}` : base;
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

export function listRunners(org: string): Promise<RunnerListResponse> {
  return request<RunnerListResponse>("GET", runnersPath(org));
}

export function createRegistrationToken(
  org: string,
): Promise<RegistrationTokenResponse> {
  return request<RegistrationTokenResponse>(
    "POST",
    `${runnersPath(org)}/registration-token`,
  );
}

export function registerRunner(
  org: string,
  req: RegisterRunnerRequest,
): Promise<Runner> {
  return request<Runner>("POST", runnersPath(org), req);
}

export function deleteRunner(org: string, runnerId: string): Promise<void> {
  return request<void>("DELETE", runnersPath(org, runnerId));
}
