import { describe, it, expect, beforeEach, vi } from "vitest";

import { API_TOKEN_KEY, ApiClient } from "./api";

describe("TestSetToken", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("setToken stores in localStorage", () => {
    const client = new ApiClient("http://localhost:8080");

    client.setToken("test-token-123");

    expect(localStorage.getItem(API_TOKEN_KEY)).toBe("test-token-123");
    expect(client.getToken()).toBe("test-token-123");
  });
});

describe("TestGetRequest", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.unstubAllGlobals();
  });

  it("fetch called with correct Authorization header", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: {
        get: (name: string) =>
          name === "content-type" ? "application/json" : null,
      },
      json: async () => ({ id: 1 }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = new ApiClient("http://localhost:8080");
    client.setToken("test-bearer-token");

    await client.get("/user");

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/user",
      expect.objectContaining({
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          Authorization: "Bearer test-bearer-token",
        },
      }),
    );
  });
});

describe("TestUnauthorizedRedirect", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.unstubAllGlobals();
  });

  it('401 response calls router.push("/login")', async () => {
    const push = vi.fn();
    const router = { push };
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      statusText: "Unauthorized",
      json: async () => ({ message: "Unauthorized" }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = new ApiClient("http://localhost:8080", router);
    client.setToken("expired-token");

    await expect(client.get("/user")).rejects.toMatchObject({
      status: 401,
      message: "Unauthorized",
    });

    expect(push).toHaveBeenCalledWith("/login");
    expect(client.getToken()).toBeNull();
    expect(localStorage.getItem(API_TOKEN_KEY)).toBeNull();
  });
});

describe("sshKeys", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.unstubAllGlobals();
  });

  it("list returns array on 200", async () => {
    const keys = [
      {
        id: "1",
        title: "My Key",
        key_type: "ssh-ed25519",
        fingerprint: "SHA256:abc",
        created_at: "2026-01-01T00:00:00Z",
        last_used_at: null,
      },
    ];
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: {
        get: (name: string) =>
          name === "content-type" ? "application/json" : null,
      },
      json: async () => keys,
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = new ApiClient("http://localhost:8080");
    const result = await client.sshKeys.list();

    expect(result).toEqual(keys);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v3/user/keys",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("create returns key object on 201", async () => {
    const key = {
      id: "2",
      title: "Work Laptop",
      key_type: "ssh-ed25519",
      fingerprint: "SHA256:xyz",
      created_at: "2026-06-01T00:00:00Z",
      last_used_at: null,
    };
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 201,
      headers: {
        get: (name: string) =>
          name === "content-type" ? "application/json" : null,
      },
      json: async () => key,
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = new ApiClient("http://localhost:8080");
    const result = await client.sshKeys.create(
      "Work Laptop",
      "ssh-ed25519 AAAA...",
    );

    expect(result).toEqual(key);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v3/user/keys",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          title: "Work Laptop",
          key: "ssh-ed25519 AAAA...",
        }),
      }),
    );
  });

  it("create throws ApiError with status 409 on conflict", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 409,
      statusText: "Conflict",
      json: async () => ({ message: "Key already exists" }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = new ApiClient("http://localhost:8080");

    await expect(
      client.sshKeys.create("Dup", "ssh-ed25519 AAAA..."),
    ).rejects.toMatchObject({
      status: 409,
    });
  });

  it("remove resolves without error on 204", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
      headers: {
        get: () => null,
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = new ApiClient("http://localhost:8080");

    await expect(client.sshKeys.remove("key-id-1")).resolves.toBeUndefined();
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v3/user/keys/key-id-1",
      expect.objectContaining({ method: "DELETE" }),
    );
  });
});

describe("users", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.unstubAllGlobals();
  });

  it("getCurrent calls GET /api/v3/user with auth header", async () => {
    const user = { id: 1, login: "alice", email: "alice@example.com" };
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: {
        get: (name: string) =>
          name === "content-type" ? "application/json" : null,
      },
      json: async () => user,
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = new ApiClient("http://localhost:8080");
    client.setToken("test-bearer-token");

    const result = await client.users.getCurrent();

    expect(result).toEqual(user);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v3/user",
      expect.objectContaining({
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          Authorization: "Bearer test-bearer-token",
        },
      }),
    );
  });

  it("updateCurrent({name:'Bob'}) calls PATCH /api/v3/user", async () => {
    const updatedUser = {
      id: 1,
      login: "alice",
      email: "alice@example.com",
      name: "Bob",
    };
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: {
        get: (name: string) =>
          name === "content-type" ? "application/json" : null,
      },
      json: async () => updatedUser,
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = new ApiClient("http://localhost:8080");
    client.setToken("test-bearer-token");

    const result = await client.users.updateCurrent({ name: "Bob" });

    expect(result).toEqual(updatedUser);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v3/user",
      expect.objectContaining({
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          Authorization: "Bearer test-bearer-token",
        },
        body: JSON.stringify({ name: "Bob" }),
      }),
    );
  });
});

describe("tokens", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.unstubAllGlobals();
  });

  it("tokens.list() calls GET /api/v3/user/tokens", async () => {
    const tokens = [
      {
        id: 1,
        note: "ci-token",
        scopes: ["repo"],
        expires_at: "2026-12-31T00:00:00Z",
        created_at: "2026-01-01T00:00:00Z",
        last_used_at: null,
        revoked_at: null,
      },
    ];
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: {
        get: (name: string) =>
          name === "content-type" ? "application/json" : null,
      },
      json: async () => tokens,
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = new ApiClient("http://localhost:8080");
    client.setToken("test-bearer-token");

    const result = await client.tokens.list();

    expect(result).toEqual(tokens);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v3/user/tokens",
      expect.objectContaining({
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          Authorization: "Bearer test-bearer-token",
        },
      }),
    );
  });

  it("tokens.revoke(5) calls DELETE /api/v3/user/tokens/5", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
      headers: {
        get: () => null,
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = new ApiClient("http://localhost:8080");
    client.setToken("test-bearer-token");

    await expect(client.tokens.revoke(5)).resolves.toBeUndefined();
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v3/user/tokens/5",
      expect.objectContaining({
        method: "DELETE",
        headers: {
          "Content-Type": "application/json",
          Authorization: "Bearer test-bearer-token",
        },
      }),
    );
  });
});
