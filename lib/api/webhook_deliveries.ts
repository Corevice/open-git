import { API_TOKEN_KEY, ApiError } from "../api";
import { env } from "../env";

export interface WebhookDelivery {
  id: string;
  event: string;
  status: string;
  status_code: number | null;
  delivered_at: string | null;
  duration_ms: number | null;
  redelivery: boolean;
}

export interface WebhookDeliveryDetail extends WebhookDelivery {
  request_headers: Record<string, string | string[]>;
  request_body: string;
  response_headers: Record<string, string | string[]> | null;
  response_body: string | null;
}

export interface WebhookDeliveryListResponse {
  deliveries: WebhookDelivery[];
  total_count?: number;
}

type HttpMethod = "GET" | "POST";

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

function deliveriesPath(
  owner: string,
  repo: string,
  hookId: number,
  deliveryId?: string,
  suffix?: string,
): string {
  const base = `/api/v3/repos/${owner}/${repo}/hooks/${hookId}/deliveries`;
  if (deliveryId === undefined) {
    return base;
  }
  const deliveryBase = `${base}/${deliveryId}`;
  return suffix ? `${deliveryBase}/${suffix}` : deliveryBase;
}

function hooksPath(owner: string, repo: string, hookId: number): string {
  return `/api/v3/repos/${owner}/${repo}/hooks/${hookId}`;
}

async function request<T>(
  method: HttpMethod,
  path: string,
): Promise<T> {
  const response = await fetch(`${resolveBaseUrl()}${path}`, {
    method,
    headers: buildHeaders(),
  });

  if (!response.ok) {
    let message = response.statusText;
    try {
      const errorBody = await response.json();
      message = (errorBody as { message?: string }).message ?? message;
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

export function listDeliveries(
  owner: string,
  repo: string,
  hookId: number,
  page = 1,
): Promise<WebhookDelivery[]> {
  const query = page > 1 ? `?page=${page}` : "";
  return request<WebhookDelivery[]>(
    "GET",
    `${deliveriesPath(owner, repo, hookId)}${query}`,
  );
}

export function getDelivery(
  owner: string,
  repo: string,
  hookId: number,
  deliveryId: string,
): Promise<WebhookDeliveryDetail> {
  return request<WebhookDeliveryDetail>(
    "GET",
    deliveriesPath(owner, repo, hookId, deliveryId),
  );
}

export function redeliverDelivery(
  owner: string,
  repo: string,
  hookId: number,
  deliveryId: string,
): Promise<void> {
  return request<void>(
    "POST",
    deliveriesPath(owner, repo, hookId, deliveryId, "attempts"),
  );
}

export function pingWebhook(
  owner: string,
  repo: string,
  hookId: number,
): Promise<void> {
  return request<void>("POST", `${hooksPath(owner, repo, hookId)}/pings`);
}

export function isSuccessStatusCode(code: number | null): boolean {
  return code !== null && code >= 200 && code < 300;
}

export const RESPONSE_BODY_TRUNCATION_BYTES = 64 * 1024;

export function formatHeaders(
  headers: Record<string, string | string[]> | null | undefined,
): string {
  if (!headers) return "";
  return Object.entries(headers)
    .map(([key, value]) => {
      const formatted = Array.isArray(value) ? value.join(", ") : value;
      return `${key}: ${formatted}`;
    })
    .join("\n");
}

export function formatBody(body: string | null | undefined): {
  text: string;
  truncated: boolean;
} {
  if (!body) {
    return { text: "", truncated: false };
  }

  const encoder = new TextEncoder();
  const bytes = encoder.encode(body);
  const truncated = bytes.length > RESPONSE_BODY_TRUNCATION_BYTES;
  const slice = truncated
    ? bytes.slice(0, RESPONSE_BODY_TRUNCATION_BYTES)
    : bytes;
  const raw = new TextDecoder().decode(slice);

  try {
    const parsed = JSON.parse(raw) as unknown;
    return {
      text: JSON.stringify(parsed, null, 2),
      truncated,
    };
  } catch {
    return { text: raw, truncated };
  }
}
