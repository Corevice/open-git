import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { SWRConfig } from "swr";
import { describe, expect, it, vi, beforeEach } from "vitest";

import {
  sanitizeRepoPath,
  sanitizeRepoRef,
  useRepoContents,
} from "@/lib/hooks/useRepoContents";

const { mockGetContents } = vi.hoisted(() => ({
  mockGetContents: vi.fn(),
}));

vi.mock("@/lib/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api-client")>();
  return {
    ...actual,
    createRepoApiClient: () => ({
      getContents: mockGetContents,
    }),
  };
});

vi.mock("@/lib/env", () => ({
  env: { NEXT_PUBLIC_API_BASE_URL: "http://test" },
}));

function createWrapper() {
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
        {children}
      </SWRConfig>
    );
  };
}

describe("sanitizeRepoPath", () => {
  it("removes traversal segments", () => {
    expect(sanitizeRepoPath("src/secret")).toBe("src/secret");
    expect(sanitizeRepoPath("")).toBe("");
  });

  it("rejects traversal and encoded traversal segments", () => {
    expect(sanitizeRepoPath("../etc/passwd")).toBeNull();
    expect(sanitizeRepoPath("src/../secret")).toBeNull();
    expect(sanitizeRepoPath("%2e%2e/secret")).toBeNull();
    expect(sanitizeRepoPath("src/%2e%2e/secret")).toBeNull();
  });
});

describe("sanitizeRepoRef", () => {
  it("accepts valid refs including slash-separated branch names", () => {
    expect(sanitizeRepoRef("main")).toBe("main");
    expect(sanitizeRepoRef("feature/foo")).toBe("feature/foo");
  });

  it("rejects refs with traversal or backslashes", () => {
    expect(sanitizeRepoRef("../main")).toBe("");
    expect(sanitizeRepoRef("foo/..")).toBe("");
    expect(sanitizeRepoRef("branch\\name")).toBe("");
    expect(sanitizeRepoRef("%2e%2e")).toBe("");
  });
});

describe("useRepoContents", () => {
  beforeEach(() => {
    mockGetContents.mockReset();
  });

  it("fetches contents with sanitized path and ref", async () => {
    mockGetContents.mockResolvedValue([{ name: "README.md", type: "file" }]);

    const { result } = renderHook(
      () => useRepoContents("owner", "repo", "../src", " main "),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(mockGetContents).not.toHaveBeenCalled();
    expect(result.current.data).toBeUndefined();
    expect(result.current.error).toBeNull();
    expect(result.current.isNotFound).toBe(false);
  });

  it("fetches contents for valid path and ref", async () => {
    mockGetContents.mockResolvedValue([{ name: "README.md", type: "file" }]);

    const { result } = renderHook(
      () => useRepoContents("owner", "repo", "src/app", "feature/foo"),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(mockGetContents).toHaveBeenCalledWith("owner", "repo", "src/app", "feature/foo");
    expect(result.current.data).toEqual([{ name: "README.md", type: "file" }]);
    expect(result.current.error).toBeNull();
    expect(result.current.isNotFound).toBe(false);
  });

  it("skips fetch when path is null", async () => {
    const { result } = renderHook(
      () => useRepoContents("owner", "repo", null, "main"),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(mockGetContents).not.toHaveBeenCalled();
    expect(result.current.data).toBeUndefined();
    expect(result.current.error).toBeNull();
  });

  it("skips fetch when path sanitizes to invalid", async () => {
    const { result } = renderHook(
      () => useRepoContents("owner", "repo", "..", "main"),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(mockGetContents).not.toHaveBeenCalled();
    expect(result.current.data).toBeUndefined();
    expect(result.current.error).toBeNull();
  });

  it("surfaces 404 as isNotFound", async () => {
    mockGetContents.mockRejectedValue({ status: 404, message: "Not found" });

    const { result } = renderHook(
      () => useRepoContents("owner", "repo", "missing.txt", "main"),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.isNotFound).toBe(true);
    });

    expect(result.current.data).toBeUndefined();
    expect(result.current.error).toEqual({ status: 404, message: "Not found" });
  });
});
