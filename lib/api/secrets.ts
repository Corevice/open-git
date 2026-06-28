import sodium from "libsodium-wrappers";
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

interface PaginatedSecretsResponse<T> {
  total_count: number;
  secrets: T[];
}

const SECRETS_PAGE_SIZE = 100;

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

function validatePathSegment(value: string, label: string): void {
  if (
    !value ||
    value.includes("/") ||
    value.includes("\\") ||
    value === "." ||
    value === ".."
  ) {
    throw new ApiError(400, `Invalid ${label}`);
  }
}

function repoSecretsPath(owner: string, repo: string, name?: string): string {
  validatePathSegment(owner, "owner");
  validatePathSegment(repo, "repo");
  if (name !== undefined) {
    validatePathSegment(name, "secret name");
  }
  const base = `/api/v3/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/actions/secrets`;
  return name !== undefined ? `${base}/${encodeURIComponent(name)}` : base;
}

function orgSecretsPath(owner: string, name?: string): string {
  validatePathSegment(owner, "owner");
  if (name !== undefined) {
    validatePathSegment(name, "secret name");
  }
  const base = `/api/v3/orgs/${encodeURIComponent(owner)}/actions/secrets`;
  return name !== undefined ? `${base}/${encodeURIComponent(name)}` : base;
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

async function fetchAllSecrets<T>(basePath: string): Promise<T[]> {
  let page = 1;
  const secrets: T[] = [];
  let totalCount = 0;

  do {
    const separator = basePath.includes("?") ? "&" : "?";
    const data = await request<PaginatedSecretsResponse<T>>(
      "GET",
      `${basePath}${separator}per_page=${SECRETS_PAGE_SIZE}&page=${page}`,
    );
    secrets.push(...data.secrets);
    totalCount = data.total_count;
    page += 1;
  } while (secrets.length < totalCount);

  return secrets;
}

export function listRepoSecrets(
  owner: string,
  repo: string,
): Promise<ActionSecret[]> {
  return fetchAllSecrets<ActionSecret>(repoSecretsPath(owner, repo));
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

export function upsertRepoSecret(
  owner: string,
  repo: string,
  name: string,
  encrypted_value: string,
  key_id: string,
): Promise<void> {
  return request<void>("PUT", repoSecretsPath(owner, repo, name), {
    encrypted_value,
    key_id,
  });
}

export function deleteRepoSecret(
  owner: string,
  repo: string,
  name: string,
): Promise<void> {
  return request<void>("DELETE", repoSecretsPath(owner, repo, name));
}

export function listOrgSecrets(owner: string): Promise<OrgActionSecret[]> {
  return fetchAllSecrets<OrgActionSecret>(orgSecretsPath(owner));
}

export function getOrgPublicKey(owner: string): Promise<PublicKey> {
  return request<PublicKey>("GET", `${orgSecretsPath(owner)}/public-key`);
}

export function upsertOrgSecret(
  owner: string,
  name: string,
  encrypted_value: string | undefined,
  key_id: string | undefined,
  visibility: SecretVisibility,
  selected_repository_ids?: number[],
): Promise<void> {
  const payload: {
    visibility: SecretVisibility;
    encrypted_value?: string;
    key_id?: string;
    selected_repository_ids?: number[];
  } = { visibility };

  if (encrypted_value !== undefined && key_id !== undefined) {
    payload.encrypted_value = encrypted_value;
    payload.key_id = key_id;
  }

  if (selected_repository_ids !== undefined) {
    payload.selected_repository_ids = selected_repository_ids;
  }

  return request<void>("PUT", orgSecretsPath(owner, name), payload);
}

export function deleteOrgSecret(owner: string, name: string): Promise<void> {
  return request<void>("DELETE", orgSecretsPath(owner, name));
}

export async function sealSecret(
  value: string,
  publicKey: PublicKey,
): Promise<{ encrypted_value: string; key_id: string }> {
  await sodium.ready;
  const base64Variant = sodium.base64_variants.ORIGINAL;
  const publicKeyBytes = sodium.from_base64(publicKey.key, base64Variant);
  const valueBytes = sodium.from_string(value);
  const sealed = sodium.crypto_box_seal(valueBytes, publicKeyBytes);

  return {
    encrypted_value: sodium.to_base64(sealed, base64Variant),
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
