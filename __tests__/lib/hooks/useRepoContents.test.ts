import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import useSWR from "swr";

import { useRepoContents } from "@/lib/hooks/useRepoContents";

vi.mock("swr", () => ({ default: vi.fn() }));

describe("useRepoContents", () => {
  it("returns mocked contents data", () => {
    vi.mocked(useSWR).mockReturnValue({
      data: [{ name: "README.md" }],
      isLoading: false,
      error: null,
    } as ReturnType<typeof useSWR>);

    const { result } = renderHook(() =>
      useRepoContents("owner", "repo", "", "main"),
    );

    expect(result.current.data).toEqual([{ name: "README.md" }]);
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();
  });
});
