import { beforeEach, describe, expect, it, vi } from "vitest";
import sodium from "libsodium-wrappers";

import { ApiError } from "../api";
import {
  deleteOrgSecret,
  getOrgPublicKey,
  getRepoPublicKey,
  isSecretValidationError,
  listOrgSecrets,
  listRepoSecrets,
  sealSecret,
  upsertOrgSecret,
  upsertRepoSecret,
  validateSecretName,
  visibilityLabel,
} from "./secrets";

describe("visibilityLabel", () => {
  it("returns human-readable labels", () => {
    expect(visibilityLabel("all")).toBe("All repositories");
    expect(visibilityLabel("private")).toBe("Private repositories");
    expect(visibilityLabel("selected")).toBe("Selected repositories");
  });
});

describe("validateSecretName", () => {
  it("accepts valid secret names", () => {
    expect(validateSecretName("MY_SECRET")).toBeNull();
    expect(validateSecretName("A")).toBeNull();
  });

  it("rejects empty names", () => {
    expect(validateSecretName("")).toBe("Secret name is required");
    expect(validateSecretName("   ")).toBe("Secret name is required");
  });

  it("rejects invalid patterns", () => {
    expect(validateSecretName("my_secret")).toBe(
      "Secret name must match [A-Z_][A-Z0-9_]*",
    );
    expect(validateSecretName("1SECRET")).toBe(
      "Secret name must match [A-Z_][A-Z0-9_]*",
    );
  });

  it("rejects GITHUB_ prefix", () => {
    expect(validateSecretName("GITHUB_TOKEN")).toBe(
      "Secret names with GITHUB_ prefix are reserved",
    );
  });
});

describe("isSecretValidationError", () => {
  it("returns true for 422 ApiError with fieldErrors", () => {
    const error = new ApiError(422, "Validation failed") as ApiError & {
      fieldErrors: Record<string, string>;
    };
    error.fieldErrors = { name: "名前は必須です" };

    expect(isSecretValidationError(error)).toBe(true);
  });

  it("returns false for non-422 ApiError", () => {
    expect(isSecretValidationError(new ApiError(401, "Unauthorized"))).toBe(
      false,
    );
  });

  it("returns false for non-ApiError values", () => {
    expect(isSecretValidationError(new Error("boom"))).toBe(false);
    expect(isSecretValidationError(null)).toBe(false);
  });
});

describe("fetchAllSecrets pagination", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    localStorage.clear();
  });

  function mockFetchSequence(
    responses: Array<{ total_count?: number; secrets: unknown[] }>,
  ) {
    let call = 0;
    const fetchMock = vi.fn().mockImplementation(async () => {
      const payload = responses[call] ?? { total_count: 0, secrets: [] };
      call += 1;
      return {
        ok: true,
        status: 200,
        headers: {
          get: (name: string) =>
            name === "content-type" ? "application/json" : null,
        },
        json: async () => payload,
      };
    });
    vi.stubGlobal("fetch", fetchMock);
    return fetchMock;
  }

  it("returns all secrets across pages", async () => {
    const fetchMock = mockFetchSequence([
      {
        total_count: 2,
        secrets: [{ name: "ONE", visibility: "all" }],
      },
      {
        total_count: 2,
        secrets: [{ name: "TWO", visibility: "private" }],
      },
    ]);

    const secrets = await listOrgSecrets("my-org");

    expect(secrets).toHaveLength(2);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("stops when response has no secrets array", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: {
        get: (name: string) =>
          name === "content-type" ? "application/json" : null,
      },
      json: async () => ({ total_count: 100 }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const secrets = await listOrgSecrets("my-org");

    expect(secrets).toEqual([]);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("stops when a page returns no secrets", async () => {
    mockFetchSequence([
      {
        total_count: 10,
        secrets: [{ name: "ONE", visibility: "all" }],
      },
      {
        total_count: 10,
        secrets: [],
      },
    ]);

    const secrets = await listOrgSecrets("my-org");

    expect(secrets).toHaveLength(1);
  });

  it("stops after a partial page when total_count is missing", async () => {
    const fetchMock = mockFetchSequence([
      {
        secrets: [{ name: "ONLY", created_at: "2024-01-01T00:00:00Z" }],
      },
    ]);

    const secrets = await listRepoSecrets("owner", "repo");

    expect(secrets).toHaveLength(1);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});

describe("validatePathSegment", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    localStorage.clear();
  });

  it("rejects encoded path traversal in owner", async () => {
    await expect(listOrgSecrets("%2e%2e")).rejects.toMatchObject({
      status: 400,
      message: "Invalid owner",
    });
  });

  it("rejects encoded slashes in repo name", async () => {
    await expect(listRepoSecrets("owner", "repo%2fadmin")).rejects.toMatchObject(
      {
        status: 400,
        message: "Invalid repo",
      },
    );
  });

  it("rejects unicode characters in owner", async () => {
    await expect(listOrgSecrets("org\u0000name")).rejects.toMatchObject({
      status: 400,
      message: "Invalid owner",
    });
  });

  it("rejects non-allowlisted characters in secret name", async () => {
    await expect(deleteOrgSecret("my-org", "bad secret")).rejects.toMatchObject(
      {
        status: 400,
        message: "Invalid secret name",
      },
    );
  });
});

describe("request error handling", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    localStorage.clear();
  });

  it("uses generic messages for non-422 errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        statusText: "Internal Server Error",
        json: async () => ({ message: "database connection leaked" }),
      }),
    );

    await expect(deleteOrgSecret("my-org", "SECRET")).rejects.toMatchObject({
      status: 500,
      message: "Request failed",
    });
  });

  it("uses generic message for 422 errors while preserving field errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 422,
        statusText: "Unprocessable Entity",
        json: async () => ({
          message: "internal validation detail leaked",
          errors: [{ field: "name", message: "Name is invalid" }],
        }),
      }),
    );

    let caught: unknown;
    try {
      await upsertRepoSecret("owner", "repo", "SECRET", "encrypted", "key-1");
    } catch (error) {
      caught = error;
    }

    expect(caught).toMatchObject({
      status: 422,
      message: "Validation failed",
    });
    expect(isSecretValidationError(caught)).toBe(true);
    if (isSecretValidationError(caught)) {
      expect(caught.fieldErrors).toEqual({ name: "Name is invalid" });
    }
  });
});

describe("upsertRepoSecret", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    localStorage.clear();
  });

  it("sends encrypted value and key id", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
      headers: { get: () => null },
    });
    vi.stubGlobal("fetch", fetchMock);

    await upsertRepoSecret("owner", "repo", "MY_SECRET", "encrypted", "key-1");

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/repos/owner/repo/actions/secrets/MY_SECRET"),
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({
          encrypted_value: "encrypted",
          key_id: "key-1",
        }),
      }),
    );
  });
});

describe("upsertOrgSecret", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    localStorage.clear();
  });

  it("sends encrypted value and visibility", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
      headers: { get: () => null },
    });
    vi.stubGlobal("fetch", fetchMock);

    await upsertOrgSecret("my-org", "MY_SECRET", "encrypted", "key-1", "all");

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/orgs/my-org/actions/secrets/MY_SECRET"),
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({
          visibility: "all",
          encrypted_value: "encrypted",
          key_id: "key-1",
        }),
      }),
    );
  });

  it("allows visibility-only updates without encrypted value", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
      headers: { get: () => null },
    });
    vi.stubGlobal("fetch", fetchMock);

    await upsertOrgSecret(
      "my-org",
      "MY_SECRET",
      undefined,
      undefined,
      "private",
    );

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/orgs/my-org/actions/secrets/MY_SECRET"),
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({ visibility: "private" }),
      }),
    );
  });

  it("rejects selected visibility without repository ids", async () => {
    await expect(
      upsertOrgSecret("my-org", "MY_SECRET", "encrypted", "key-1", "selected"),
    ).rejects.toMatchObject({
      status: 400,
      message: "Selected repositories requires at least one repository",
    });
  });

  it("includes selected repository ids when provided", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
      headers: { get: () => null },
    });
    vi.stubGlobal("fetch", fetchMock);

    await upsertOrgSecret(
      "my-org",
      "MY_SECRET",
      "encrypted",
      "key-1",
      "selected",
      [1, 2],
    );

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/orgs/my-org/actions/secrets/MY_SECRET"),
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({
          visibility: "selected",
          encrypted_value: "encrypted",
          key_id: "key-1",
          selected_repository_ids: [1, 2],
        }),
      }),
    );
  });
});

describe("getOrgPublicKey", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    localStorage.clear();
  });

  it("rejects invalid public key responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: {
          get: (name: string) =>
            name === "content-type" ? "application/json" : null,
        },
        json: async () => ({ key_id: "bad-key", key: "not-valid-base64!!!" }),
      }),
    );

    await expect(getOrgPublicKey("my-org")).rejects.toMatchObject({
      status: 400,
      message: "Invalid public key",
    });
  });

  it("returns validated public keys", async () => {
    await sodium.ready;
    const keyPair = sodium.crypto_box_keypair();
    const base64Variant = sodium.base64_variants.ORIGINAL;

    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: {
          get: (name: string) =>
            name === "content-type" ? "application/json" : null,
        },
        json: async () => ({
          key_id: "test-key-id",
          key: sodium.to_base64(keyPair.publicKey, base64Variant),
        }),
      }),
    );

    await expect(getOrgPublicKey("my-org")).resolves.toMatchObject({
      key_id: "test-key-id",
    });
  });
});

describe("getRepoPublicKey", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    localStorage.clear();
  });

  it("rejects public keys with incorrect byte length", async () => {
    await sodium.ready;
    const base64Variant = sodium.base64_variants.ORIGINAL;
    const shortKey = sodium.to_base64(new Uint8Array(16), base64Variant);

    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: {
          get: (name: string) =>
            name === "content-type" ? "application/json" : null,
        },
        json: async () => ({ key_id: "short-key", key: shortKey }),
      }),
    );

    await expect(getRepoPublicKey("owner", "repo")).rejects.toMatchObject({
      status: 400,
      message: "Invalid public key",
    });
  });
});

describe("sealSecret", () => {
  it("returns encrypted value and key id", async () => {
    await sodium.ready;
    const keyPair = sodium.crypto_box_keypair();
    const base64Variant = sodium.base64_variants.ORIGINAL;
    const publicKey = {
      key_id: "test-key-id",
      key: sodium.to_base64(keyPair.publicKey, base64Variant),
    };

    const result = await sealSecret("super-secret", publicKey);

    expect(result.key_id).toBe("test-key-id");
    expect(result.encrypted_value).toBeTruthy();
    expect(result.encrypted_value).not.toBe("super-secret");
  });
});
