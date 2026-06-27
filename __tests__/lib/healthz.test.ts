import { beforeEach, describe, expect, it, vi } from "vitest";

import { checkHealth } from "@/lib/healthz";

describe("checkHealth", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    vi.unstubAllEnvs();
    vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "http://localhost:8080");
  });

  it("returns HealthResponse on success", async () => {
    const health = {
      status: "ok",
      version: "0.1.0",
      time: "2025-01-01T00:00:00Z",
    };
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => health,
    });
    vi.stubGlobal("fetch", fetchMock);

    const result = await checkHealth();

    expect(result).toEqual(health);
    expect(fetchMock).toHaveBeenCalledWith("http://localhost:8080/healthz");
  });

  it("returns null when fetch rejects", async () => {
    const fetchMock = vi.fn().mockRejectedValue(new Error("network error"));
    vi.stubGlobal("fetch", fetchMock);

    const result = await checkHealth();

    expect(result).toBeNull();
  });
});
