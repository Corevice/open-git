import { beforeEach, describe, expect, it, vi } from "vitest";

import { createApiClient } from "@/lib/api-client";

describe("api-client patch", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("patch sends correct method, body, and Authorization header", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ state: "closed" }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = createApiClient("http://localhost:8080");
    await client.patch(
      "/api/v3/repos/acme/demo/issues/1",
      { state: "closed" },
      { token: "test-token" },
    );

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v3/repos/acme/demo/issues/1",
      expect.objectContaining({
        method: "PATCH",
        headers: expect.objectContaining({
          Authorization: "Bearer test-token",
          Accept: "application/vnd.github+json",
          "Content-Type": "application/json",
        }),
        body: JSON.stringify({ state: "closed" }),
      }),
    );
  });

  it("403 response throws ApiError with status 403", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 403,
      statusText: "Forbidden",
      json: async () => ({ message: "Forbidden" }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = createApiClient("http://localhost:8080");

    await expect(
      client.patch("/api/v3/repos/acme/demo/issues/1", { state: "closed" }),
    ).rejects.toMatchObject({
      status: 403,
      code: "forbidden",
    });
  });

  it("409 response throws ApiError with status 409", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 409,
      statusText: "Conflict",
      json: async () => ({ message: "Conflict" }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = createApiClient("http://localhost:8080");

    await expect(
      client.patch("/api/v3/repos/acme/demo/issues/1", { state: "closed" }),
    ).rejects.toMatchObject({
      status: 409,
      code: "conflict",
    });
  });

  it("401 response triggers window.location.href = '/sign-in'", async () => {
    const location = { href: "" };
    vi.stubGlobal("window", { location });

    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      statusText: "Unauthorized",
      json: async () => ({ message: "Unauthorized" }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = createApiClient("http://localhost:8080");

    await expect(
      client.patch("/api/v3/repos/acme/demo/issues/1", { state: "closed" }),
    ).rejects.toMatchObject({ status: 401 });

    expect(location.href).toBe("/sign-in");
  });
});
