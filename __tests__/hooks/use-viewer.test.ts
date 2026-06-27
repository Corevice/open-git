import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ReactNode } from "react";
import { apiClient } from "@/lib/api-client";
import { AuthProvider } from "@/components/providers/auth-provider";
import { useViewer } from "@/hooks/use-viewer";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
}));

const mockViewer = {
  id: 1,
  login: "octocat",
  name: "Octo Cat",
  avatarUrl: "https://example.com/a.png",
};

function createWrapper(token: string | null) {
  vi.spyOn(Storage.prototype, "getItem").mockImplementation((key) => {
    if (key === "pat" && token) {
      return token;
    }
    return null;
  });

  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <AuthProvider>{children}</AuthProvider>
      </QueryClientProvider>
    );
  };
}

describe("useViewer", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("returns viewer data on success", async () => {
    vi.spyOn(apiClient, "get").mockResolvedValue(mockViewer);

    const { result } = renderHook(() => useViewer(), {
      wrapper: createWrapper("test-token"),
    });

    await waitFor(() => {
      expect(result.current.viewer).toEqual(mockViewer);
    });
  });

  it("returns viewer null when token is null", () => {
    const getSpy = vi.spyOn(apiClient, "get");

    const { result } = renderHook(() => useViewer(), {
      wrapper: createWrapper(null),
    });

    expect(result.current.viewer).toBeNull();
    expect(getSpy).not.toHaveBeenCalled();
  });
});
