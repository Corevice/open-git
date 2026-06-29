import { beforeEach, describe, expect, it, vi } from "vitest";

import { createApiClient, createRepoApiClient } from "@/lib/api-client";

describe("api-client", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("get() with token includes Authorization header", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: async () => JSON.stringify({ id: 1 }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = createApiClient("http://localhost:8080");
    await client.get("/user", { token: "test-token" });

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/user",
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: "Bearer test-token",
          Accept: "application/vnd.github+json",
        }),
      }),
    );
  });

  it("get() without token omits Authorization header", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: async () => JSON.stringify({ id: 1 }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = createApiClient("http://localhost:8080");
    await client.get("/user");

    const headers = fetchMock.mock.calls[0][1].headers as Record<
      string,
      string
    >;
    expect(headers.Authorization).toBeUndefined();
    expect(headers.Accept).toBe("application/vnd.github+json");
  });

  it("400 response throws ApiError with status 400", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 400,
      statusText: "Bad Request",
      json: async () => ({ code: "bad_request", message: "Invalid input" }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = createApiClient("http://localhost:8080");

    await expect(client.get("/bad")).rejects.toMatchObject({
      status: 400,
      code: "bad_request",
      message: "Invalid input",
    });
  });

  it("200 response returns parsed JSON", async () => {
    const data = { status: "ok", version: "0.1.0" };
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: async () => JSON.stringify(data),
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = createApiClient("http://localhost:8080");
    const result = await client.get("/healthz");

    expect(result).toEqual(data);
  });

  it("createRef with token includes Authorization header", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
      text: async () => "",
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = createRepoApiClient("http://localhost:8080");
    await client.createRef(
      "owner",
      "repo",
      "refs/heads/feature",
      "abc123",
      { token: "test-token" },
    );

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/repos/owner/repo/git/refs",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({
          Authorization: "Bearer test-token",
          Accept: "application/vnd.github+json",
        }),
      }),
    );
  });
});
