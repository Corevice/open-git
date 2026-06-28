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
// GitHub paginates at 100/page; cap at 100 pages (10,000 secrets) to avoid unbounded fetches.
const SECRETS_MAX_PAGES = 100;
const SECRET_NAME_PATTERN = /^[A-Z_][A-Z0-9_]*$/;
const PATH_SEGMENT_PATTERN = /^[a-zA-Z0-9._-]+$/;

let cachedBaseUrl: string | undefined;
let sodiumReady: Promise<void> | undefined;

function ensureSodiumReady(): Promise<void> {
  if (!sodiumReady) {
    sodiumReady = sodium.ready;
  }
  return sodiumReady;
}

export function visibilityLabel(visibility: SecretVisibility): string {
  switch (visibility) {
    case "all":
      return "All repositories";
    case "private":
      return "Private repositories";
    case "selected":
      return "Selected repositories";
  }
}

function resolveBaseUrl(): string {
  if (cachedBaseUrl === undefined) {
    cachedBaseUrl = env.NEXT_PUBLIC_API_BASE_URL.replace(/\/$/, "");
  }
  return cachedBaseUrl;
}

export function validateSecretName(name: string): string | null {
  const trimmed = name.trim();
  if (!trimmed) {
    return "Secret name is required";
  }
  if (!SECRET_NAME_PATTERN.test(trimmed)) {
    return "Secret name must match [A-Z_][A-Z0-9_]*";
  }
  if (trimmed.startsWith("GITHUB_")) {
    return "Secret names with GITHUB_ prefix are reserved";
  }
  return null;
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
  const normalized = value.toLowerCase();
  if (
    normalized.includes("%2f") ||
    normalized.includes("%5c") ||
    normalized.includes("..")
  ) {
    throw new ApiError(400, `Invalid ${label}`);
  }

  let decoded: string;
  try {
    decoded = decodeURIComponent(value);
  } catch {
    throw new ApiError(400, `Invalid ${label}`);
  }

  if (
    !decoded ||
    decoded.includes("\0") ||
    decoded.includes("/") ||
    decoded.includes("\\") ||
    decoded === "." ||
    decoded === ".." ||
    decoded.includes("..") ||
    !PATH_SEGMENT_PATTERN.test(decoded)
  ) {
    throw new ApiError(400, `Invalid ${label}`);
  }
}

function validatePublicKey(publicKey: PublicKey): void {
  if (!publicKey?.key_id || typeof publicKey.key_id !== "string") {
    throw new ApiError(400, "Invalid public key");
  }
  if (!publicKey?.key || typeof publicKey.key !== "string") {
    throw new ApiError(400, "Invalid public key");
  }

  try {
    const publicKeyBytes = sodium.from_base64(
      publicKey.key,
      sodium.base64_variants.ORIGINAL,
    );
    if (publicKeyBytes.length !== sodium.crypto_box_PUBLICKEYBYTES) {
      throw new ApiError(400, "Invalid public key");
    }
  } catch (error) {
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(400, "Invalid public key");
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
        "Validation failed",
      ) as SecretValidationError;
      error.fieldErrors = fieldErrors;
      throw error;
    }

    const safeMessage =
      response.status === 401
        ? "Unauthorized"
        : response.status === 403
          ? "Forbidden"
          : response.status === 404
            ? "Not found"
            : "Request failed";
    throw new ApiError(response.status, safeMessage);
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
  const secrets: T[] = [];
  let page = 1;

  while (page <= SECRETS_MAX_PAGES) {
    const separator = basePath.includes("?") ? "&" : "?";
    const data = await request<PaginatedSecretsResponse<T>>(
      "GET",
      `${basePath}${separator}per_page=${SECRETS_PAGE_SIZE}&page=${page}`,
    );

    if (!data?.secrets || !Array.isArray(data.secrets)) {
      break;
    }

    if (data.secrets.length === 0) {
      break;
    }

    secrets.push(...data.secrets);

    if (
      typeof data.total_count === "number" &&
      data.total_count >= 0 &&
      secrets.length >= data.total_count
    ) {
      break;
    }

    if (data.secrets.length < SECRETS_PAGE_SIZE) {
      break;
    }

    page += 1;
  }

  return secrets;
}

export function listRepoSecrets(
  owner: string,
  repo: string,
): Promise<ActionSecret[]> {
  return fetchAllSecrets<ActionSecret>(repoSecretsPath(owner, repo));
}

export async function getRepoPublicKey(
  owner: string,
  repo: string,
): Promise<PublicKey> {
  const publicKey = await request<PublicKey>(
    "GET",
    `${repoSecretsPath(owner, repo)}/public-key`,
  );
  await ensureSodiumReady();
  validatePublicKey(publicKey);
  return publicKey;
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

export async function getOrgPublicKey(owner: string): Promise<PublicKey> {
  const publicKey = await request<PublicKey>(
    "GET",
    `${orgSecretsPath(owner)}/public-key`,
  );
  await ensureSodiumReady();
  validatePublicKey(publicKey);
  return publicKey;
}

export function upsertOrgSecret(
  owner: string,
  name: string,
  encrypted_value: string | undefined,
  key_id: string | undefined,
  visibility: SecretVisibility,
  selected_repository_ids?: number[],
): Promise<void> {
  if (
    visibility === "selected" &&
    (selected_repository_ids === undefined ||
      selected_repository_ids.length === 0)
  ) {
    return Promise.reject(
      new ApiError(
        400,
        "Selected repositories requires at least one repository",
      ),
    );
  }

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
  await ensureSodiumReady();
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
