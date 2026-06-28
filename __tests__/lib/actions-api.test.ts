import { beforeEach, describe, expect, it, vi } from "vitest";

import { cancelRun, listRuns } from "@/lib/api/actions";

describe("actions-api", () => {
  const mockFetch = vi.fn();
  const mockGetItem = vi.fn();

  beforeEach(() => {
    vi.unstubAllGlobals();
    mockFetch.mockReset();
    mockGetItem.mockReset();
    mockGetItem.mockReturnValue("test-token");
    const localStorageStub = { getItem: mockGetItem };
    vi.stubGlobal("localStorage", localStorageStub);
    if (typeof window !== "undefined") {
      Object.defineProperty(window, "localStorage", {
        configurable: true,
        value: localStorageStub,
      });
    }
    vi.stubGlobal("fetch", mockFetch);
  });

  it("listRuns calls GET /api/repos/alice/demo/actions/runs", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      headers: { get: () => "application/json" },
      json: async () => ({ total_count: 0, workflow_runs: [] }),
    });

    await listRuns("alice", "demo");

    expect(mockFetch).toHaveBeenCalledWith(
      "/api/repos/alice/demo/actions/runs",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("cancelRun calls POST .../cancel", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 202,
      headers: { get: () => null },
    });

    await cancelRun("alice", "demo", 42);

    expect(mockFetch).toHaveBeenCalledWith(
      "/api/repos/alice/demo/actions/runs/42/cancel",
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("includes Authorization header", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      headers: { get: () => "application/json" },
      json: async () => ({ total_count: 0, workflow_runs: [] }),
    });

    await listRuns("alice", "demo");

    const headers = mockFetch.mock.calls[0][1].headers as Record<string, string>;
    expect(headers.Authorization).toBe("Bearer test-token");
    expect(mockGetItem).toHaveBeenCalledWith("open-git-auth-token");
  });

  it("throws on 404 response", async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 404,
      statusText: "Not Found",
    });

    await expect(listRuns("alice", "demo")).rejects.toThrow("HTTP 404: Not Found");
  });
});
