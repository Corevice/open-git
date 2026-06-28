import { API_TOKEN_KEY, ApiError } from "../api";
import { env } from "../env";

export type SecretVisibility = "all" | "private" | "selected";

export interface PublicKey {
  key_id: string;
  key: string;
}

export interface ActionSecret {
  name: string;
  created_at?: string;
  updated_at?: string;
}

export interface OrgActionSecret extends ActionSecret {
  visibility: SecretVisibility;
}

export interface SecretValidationError extends ApiError {
  fieldErrors: Record<string, string>;
}

type HttpMethod = "GET" | "PUT" | "DELETE";

interface ListSecretsResponse {
  total_count: number;
  secrets: ActionSecret[];
}

interface ListOrgSecretsResponse {
  total_count: number;
  secrets: OrgActionSecret[];
}

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

function parseFieldErrors(
  body: unknown,
): Record<string, string> | undefined {
  if (!body || typeof body !== "object") return undefined;
  const errors = (
    body as { errors?: Array<{ field?: string; message?: string }> }
  ).errors;
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
      message = (errorBody as { message?: string }).message ?? message;
    } catch {
      // ignore JSON parse errors
    }

    if (response.status === 422) {
      const fieldErrors = parseFieldErrors(errorBody) ?? {};
      const error = new ApiError(
        response.status,
        message,
      ) as SecretValidationError;
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

function encodeSecretName(name: string): string {
  return encodeURIComponent(name);
}

export function listRepoSecrets(
  owner: string,
  repo: string,
): Promise<ActionSecret[]> {
  return request<ListSecretsResponse>(
    "GET",
    `/api/v3/repos/${owner}/${repo}/actions/secrets`,
  ).then((data) => data.secrets);
}

export function getRepoPublicKey(
  owner: string,
  repo: string,
): Promise<PublicKey> {
  return request<PublicKey>(
    "GET",
    `/api/v3/repos/${owner}/${repo}/actions/secrets/public-key`,
  );
}

export function upsertRepoSecret(
  owner: string,
  repo: string,
  name: string,
  encrypted_value: string,
  key_id: string,
): Promise<void> {
  return request<void>(
    "PUT",
    `/api/v3/repos/${owner}/${repo}/actions/secrets/${encodeSecretName(name)}`,
    { encrypted_value, key_id },
  );
}

export function deleteRepoSecret(
  owner: string,
  repo: string,
  name: string,
): Promise<void> {
  return request<void>(
    "DELETE",
    `/api/v3/repos/${owner}/${repo}/actions/secrets/${encodeSecretName(name)}`,
  );
}

export function listOrgSecrets(org: string): Promise<OrgActionSecret[]> {
  return request<ListOrgSecretsResponse>(
    "GET",
    `/api/v3/orgs/${org}/actions/secrets`,
  ).then((data) => data.secrets);
}

export function getOrgPublicKey(org: string): Promise<PublicKey> {
  return request<PublicKey>(
    "GET",
    `/api/v3/orgs/${org}/actions/secrets/public-key`,
  );
}

export function upsertOrgSecret(
  org: string,
  name: string,
  encrypted_value: string,
  key_id: string,
  visibility: SecretVisibility,
  selected_repository_ids?: number[],
): Promise<void> {
  const payload: {
    encrypted_value: string;
    key_id: string;
    visibility: SecretVisibility;
    selected_repository_ids?: number[];
  } = {
    encrypted_value,
    key_id,
    visibility,
  };

  if (selected_repository_ids !== undefined) {
    payload.selected_repository_ids = selected_repository_ids;
  }

  return request<void>(
    "PUT",
    `/api/v3/orgs/${org}/actions/secrets/${encodeSecretName(name)}`,
    payload,
  );
}

export function deleteOrgSecret(org: string, name: string): Promise<void> {
  return request<void>(
    "DELETE",
    `/api/v3/orgs/${org}/actions/secrets/${encodeSecretName(name)}`,
  );
}

export async function sealSecret(
  value: string,
  publicKey: PublicKey,
): Promise<{ encrypted_value: string; key_id: string }> {
  const bytes = new TextEncoder().encode(value);
  let binary = "";
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]!);
  }

  return {
    encrypted_value: btoa(binary),
    key_id: publicKey.key_id,
  };
}

export function isSecretValidationError(
  error: unknown,
): error is SecretValidationError {
  return (
    error instanceof ApiError &&
    error.status === 422 &&
    "fieldErrors" in error
  );
}
