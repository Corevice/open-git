declare module "libsodium-wrappers" {
  interface LibSodium {
    ready: Promise<void>;
    from_base64(input: string): Uint8Array;
    from_string(input: string): Uint8Array;
    to_base64(input: Uint8Array): string;
    crypto_box_seal(
      message: Uint8Array,
      publicKey: Uint8Array,
    ): Uint8Array;
  }
  const sodium: LibSodium;
  export default sodium;
}

import sodium from "libsodium-wrappers";
import { API_TOKEN_KEY, ApiError } from "../api";
import { env } from "../env";

export interface ActionSecret {
  name: string;
  created_at: string;
  updated_at: string;
  visibility?: string;
}

export interface PublicKey {
  key_id: string;
  key: string;
}

export interface SecretValidationError extends ApiError {
  fieldErrors: Record<string, string>;
}

type HttpMethod = "GET" | "PUT" | "DELETE";

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

function repoSecretsPath(owner: string, repo: string, name?: string): string {
  const base = `/api/v3/repos/${owner}/${repo}/actions/secrets`;
  return name !== undefined ? `${base}/${encodeURIComponent(name)}` : base;
}

function orgSecretsPath(org: string, name?: string): string {
  const base = `/api/v3/orgs/${org}/actions/secrets`;
  return name !== undefined ? `${base}/${encodeURIComponent(name)}` : base;
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
      const error = new ApiError(response.status, message) as SecretValidationError;
      error.fieldErrors = fieldErrors;
      throw error;
    }

    throw new ApiError(response.status, message);
  }

  if (response.status === 204 || response.status === 201) {
    return undefined as T;
  }

  const contentType = response.headers.get("content-type");
  if (!contentType?.includes("application/json")) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}

export async function sealSecret(
  value: string,
  publicKey: PublicKey,
): Promise<{ encrypted_value: string; key_id: string }> {
  await sodium.ready;
  const publicKeyBytes = sodium.from_base64(publicKey.key);
  const valueBytes = sodium.from_string(value);
  const sealed = sodium.crypto_box_seal(valueBytes, publicKeyBytes);
  return {
    encrypted_value: sodium.to_base64(sealed),
    key_id: publicKey.key_id,
  };
}

export function getRepoPublicKey(
  owner: string,
  repo: string,
): Promise<PublicKey> {
  return request<PublicKey>(
    "GET",
    `${repoSecretsPath(owner, repo)}/public-key`,
  );
}

export function listRepoSecrets(
  owner: string,
  repo: string,
): Promise<{ total_count: number; secrets: ActionSecret[] }> {
  return request<{ total_count: number; secrets: ActionSecret[] }>(
    "GET",
    repoSecretsPath(owner, repo),
  );
}

export function getRepoSecret(
  owner: string,
  repo: string,
  name: string,
): Promise<ActionSecret> {
  return request<ActionSecret>("GET", repoSecretsPath(owner, repo, name));
}

export function upsertRepoSecret(
  owner: string,
  repo: string,
  name: string,
  encryptedValue: string,
  keyId: string,
): Promise<void> {
  return request<void>("PUT", repoSecretsPath(owner, repo, name), {
    encrypted_value: encryptedValue,
    key_id: keyId,
  });
}

export function deleteRepoSecret(
  owner: string,
  repo: string,
  name: string,
): Promise<void> {
  return request<void>("DELETE", repoSecretsPath(owner, repo, name));
}

export function listOrgSecrets(
  org: string,
): Promise<{ total_count: number; secrets: ActionSecret[] }> {
  return request<{ total_count: number; secrets: ActionSecret[] }>(
    "GET",
    orgSecretsPath(org),
  );
}

export function upsertOrgSecret(
  org: string,
  name: string,
  encryptedValue: string,
  keyId: string,
  visibility: string,
  selectedRepoIds?: number[],
): Promise<void> {
  const body: Record<string, unknown> = {
    encrypted_value: encryptedValue,
    key_id: keyId,
    visibility,
  };
  if (selectedRepoIds !== undefined) {
    body.selected_repository_ids = selectedRepoIds;
  }
  return request<void>("PUT", orgSecretsPath(org, name), body);
}

export function deleteOrgSecret(org: string, name: string): Promise<void> {
  return request<void>("DELETE", orgSecretsPath(org, name));
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
