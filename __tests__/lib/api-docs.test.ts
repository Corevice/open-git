import { beforeEach, describe, expect, it, vi } from "vitest";

import { createApiClient } from "@/lib/api-client";

describe("api-docs", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("getDocTree returns sections", async () => {
    const sections = [
      { slug: "getting-started", title: "Getting Started", order: 1 },
    ];
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: async () => JSON.stringify({ sections }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = createApiClient("http://localhost:8080");
    const result = await client.getDocTree();

    expect(result).toEqual({ sections });
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/docs/contributing",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("getDocSection returns content with content_markdown", async () => {
    const content = {
      slug: "getting-started",
      title: "Getting Started",
      content_markdown: "# Hello",
      updated_at: "2025-01-01T00:00:00Z",
      edit_url: "https://example.com/edit",
    };
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: async () => JSON.stringify(content),
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = createApiClient("http://localhost:8080");
    const result = await client.getDocSection("getting-started");

    expect(result).toEqual(content);
    expect(result.content_markdown).toBe("# Hello");
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/docs/contributing/getting-started",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("getContributors returns array", async () => {
    const contributors = [
      {
        login: "alice",
        id: 1,
        avatar_url: "https://example.com/avatar.png",
        contributions: 42,
        type: "User",
      },
    ];
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: async () => JSON.stringify(contributors),
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = createApiClient("http://localhost:8080");
    const result = await client.getContributors("alice", "repo");

    expect(result).toEqual(contributors);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/repos/alice/repo/contributors?page=1&per_page=30",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("getContributors passes page and perPage query params", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: async () => JSON.stringify([]),
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = createApiClient("http://localhost:8080");
    await client.getContributors("owner", "repo", 2, 10);

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/repos/owner/repo/contributors?page=2&per_page=10",
      expect.objectContaining({ method: "GET" }),
    );
  });
});
