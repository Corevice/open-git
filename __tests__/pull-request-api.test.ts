import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  API_TOKEN_KEY,
  createPullRequest,
  listPullRequests,
  mergePullRequest,
} from "@/lib/api";

function mockJsonResponse(data: unknown, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 200 ? "OK" : "Error",
    headers: {
      get: (name: string) =>
        name === "content-type" ? "application/json" : null,
    },
    json: async () => data,
  };
}

describe("pull request api", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.unstubAllGlobals();
  });

  it("listPullRequests calls correct URL with state param", async () => {
    const fetchMock = vi.fn().mockResolvedValue(mockJsonResponse([]));
    vi.stubGlobal("fetch", fetchMock);

    localStorage.setItem(API_TOKEN_KEY, "test-token");

    await listPullRequests("acme", "demo", "open", 2, 30);

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v3/repos/acme/demo/pulls?state=open&page=2&per_page=30",
      expect.objectContaining({
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          Authorization: "Bearer test-token",
        },
      }),
    );
  });

  it("createPullRequest sends POST body", async () => {
    const created = {
      id: "pr-1",
      number: 42,
      title: "Add feature",
      body: "Details",
      state: "open",
      draft: false,
      head_ref: "feature",
      base_ref: "main",
      head_sha: "",
      base_sha: "",
      merge_commit_sha: null,
      mergeable: true,
      mergeable_state: "clean",
      merged_at: null,
      merged_by: null,
      author_id: "user-1",
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };
    const fetchMock = vi.fn().mockResolvedValue(mockJsonResponse(created, 201));
    vi.stubGlobal("fetch", fetchMock);

    localStorage.setItem(API_TOKEN_KEY, "test-token");

    const result = await createPullRequest("acme", "demo", {
      title: "Add feature",
      head: "feature",
      base: "main",
      body: "Details",
    });

    expect(result).toEqual(created);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v3/repos/acme/demo/pulls",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          title: "Add feature",
          head: "feature",
          base: "main",
          body: "Details",
        }),
      }),
    );
  });

  it("mergePullRequest sends PUT with merge_method", async () => {
    const merged = {
      merged: true,
      message: "Pull Request successfully merged",
    };
    const fetchMock = vi.fn().mockResolvedValue(mockJsonResponse(merged));
    vi.stubGlobal("fetch", fetchMock);

    localStorage.setItem(API_TOKEN_KEY, "test-token");

    const result = await mergePullRequest("acme", "demo", 7, {
      merge_method: "squash",
    });

    expect(result).toEqual(merged);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v3/repos/acme/demo/pulls/7/merge",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({ merge_method: "squash" }),
      }),
    );
  });
});
