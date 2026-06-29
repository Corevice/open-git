import { API_TOKEN_KEY, ApiError } from "../api";
import { env } from "../env";

export interface WebhookConfig {
  url: string;
  content_type: "json" | "form";
  insecure_ssl?: string;
  secret?: string;
}

export interface Webhook {
  type?: string;
  id: number;
  name: string;
  active: boolean;
  events: string[];
  config: WebhookConfig;
  updated_at?: string;
  created_at?: string;
  last_response?: {
    code?: number | null;
    status?: string;
    message?: string;
  };
}

export interface CreateWebhookPayload {
  name?: string;
  active: boolean;
  events: string[];
  config: {
    url: string;
    content_type: "json" | "form";
    secret?: string;
  };
}

export type UpdateWebhookPayload = Partial<
  Omit<CreateWebhookPayload, "config">
> & {
  config?: Partial<CreateWebhookPayload["config"]>;
};

export interface WebhookValidationError extends ApiError {
  fieldErrors: Record<string, string>;
}

type HttpMethod = "GET" | "POST" | "PATCH" | "DELETE";

function resolveBaseUrl(): string {
  return env.NEXT_PUBLIC_API_BASE_URL.replace(/\/$/, "");
}

function buildHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    Accept: "application/vnd.github+json",
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

function hooksPath(owner: string, repo: string, hookId?: number): string {
  const base = `/api/v3/repos/${owner}/${repo}/hooks`;
  return hookId !== undefined ? `${base}/${hookId}` : base;
}

function parseFieldErrors(
  body: unknown,
): Record<string, string> | undefined {
  if (!body || typeof body !== "object") return undefined;
  const errors = (body as { errors?: Array<{ field?: string; message?: string }> })
    .errors;
  if (!Array.isArray(errors)) return undefined;

  const fieldErrors: Record<string, string> = {};
  for (const err of errors) {
    if (err.field && err.message) {
      fieldErrors[err.field] = err.message;
    }
  }
  return Object.keys(fieldErrors).length > 0 ? fieldErrors : undefined;
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
    let errorBody: unknown;
    try {
      errorBody = await response.json();
      message =
        (errorBody as { message?: string }).message ?? message;
    } catch {
      // ignore JSON parse errors
    }

    if (response.status === 422) {
      const fieldErrors = parseFieldErrors(errorBody) ?? {};
      const error = new ApiError(response.status, message) as WebhookValidationError;
      error.fieldErrors = fieldErrors;
      throw error;
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

export function listWebhooks(
  owner: string,
  repo: string,
): Promise<Webhook[]> {
  return request<Webhook[]>("GET", hooksPath(owner, repo));
}

export function createWebhook(
  owner: string,
  repo: string,
  data: CreateWebhookPayload,
): Promise<Webhook> {
  return request<Webhook>("POST", hooksPath(owner, repo), {
    name: "web",
    ...data,
  });
}

export function getWebhook(
  owner: string,
  repo: string,
  hookId: number,
): Promise<Webhook> {
  return request<Webhook>("GET", hooksPath(owner, repo, hookId));
}

export function updateWebhook(
  owner: string,
  repo: string,
  hookId: number,
  data: UpdateWebhookPayload,
): Promise<Webhook> {
  return request<Webhook>("PATCH", hooksPath(owner, repo, hookId), data);
}

export function deleteWebhook(
  owner: string,
  repo: string,
  hookId: number,
): Promise<void> {
  return request<void>("DELETE", hooksPath(owner, repo, hookId));
}

export function isWebhookValidationError(
  error: unknown,
): error is WebhookValidationError {
  return (
    error instanceof ApiError &&
    error.status === 422 &&
    "fieldErrors" in error
  );
}

export function formatEvents(events: string[]): string {
  if (events.includes("*")) {
    return "All events";
  }
  return events.join(", ");
}

export function lastDeliveryLabel(webhook: Webhook): string {
  const status = webhook.last_response?.status;
  const code = webhook.last_response?.code;
  if (status) return status;
  if (code !== undefined && code !== null) return String(code);
  return "—";
}
