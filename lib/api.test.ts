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
