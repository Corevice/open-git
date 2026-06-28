import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mockPush = vi.fn();
const mockUseAuth = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => mockUseAuth(),
}));

vi.mock("@/lib/api", () => ({
  getOrgs: vi.fn(),
  getCurrentUser: vi.fn(),
}));

import { getCurrentUser, getOrgs } from "@/lib/api";
import { Header } from "@/components/layout/Header";

describe("Header", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mockPush.mockClear();
    mockUseAuth.mockReturnValue({
      token: null,
      isAuthenticated: false,
      logout: vi.fn(),
    });
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({}),
      }),
    );
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it("renders Sign in link when unauthenticated", () => {
    render(<Header />);

    expect(screen.getByRole("link", { name: "Sign in" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "User menu" })).not.toBeInTheDocument();
  });

  it("renders avatar initials and org switcher when authenticated", async () => {
    vi.mocked(getCurrentUser).mockResolvedValue({
      login: "octocat",
      avatar_url: "https://example.com/octocat.png",
    });
    vi.mocked(getOrgs).mockResolvedValue([
      { login: "github", avatar_url: "https://example.com/github.png" },
    ]);
    mockUseAuth.mockReturnValue({
      token: "test-token",
      isAuthenticated: true,
      logout: vi.fn(),
    });

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(<Header />);

    await waitFor(() => {
      expect(getOrgs).toHaveBeenCalledWith("test-token");
    });

    expect(screen.getByText("OC")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "User menu" }));

    expect(await screen.findByText("github")).toBeInTheDocument();
  });

  it("debounces search fetch by 300ms", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({}),
    });
    vi.stubGlobal("fetch", fetchMock);

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(<Header />);

    await user.type(screen.getByPlaceholderText("Search or jump to..."), "react");

    expect(fetchMock).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(300);

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/api/v3/search/repositories?q=react"),
      expect.any(Object),
    );
  });

  it("does not navigate on empty search submit", async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(<Header />);

    const input = screen.getByPlaceholderText("Search or jump to...");
    await user.click(input);
    await user.keyboard("{Enter}");

    expect(mockPush).not.toHaveBeenCalled();
  });
});
