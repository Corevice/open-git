import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import NewPullRequestPage from "@/app/(app)/[owner]/[repo]/pulls/new/page";

const mockPush = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => ({ isAuthenticated: true, token: "test-token" }),
}));

describe("NewPullRequestPage", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    mockPush.mockClear();
    vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "http://localhost:8080");

    fetchMock = vi.fn((url: string | URL, init?: RequestInit) => {
      const urlStr = String(url);

      if (init?.method === "POST" && urlStr.includes("/api/v3/repos/acme/demo/pulls")) {
        return Promise.resolve({
          ok: true,
          status: 201,
          json: async () => ({ number: 7 }),
        });
      }

      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({}),
      });
    });

    vi.stubGlobal("fetch", fetchMock);
  });

  it("TestPullsNewRendersForm", async () => {
    render(<NewPullRequestPage params={Promise.resolve({ owner: "acme", repo: "demo" })} />);

    await waitFor(() => {
      expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
    });

    expect(screen.getByLabelText(/body/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/head branch/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/base branch/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /create pull request/i })).toBeInTheDocument();
  });

  it("TestPullsNewValidatesEmptyTitle", async () => {
    const user = userEvent.setup();

    render(<NewPullRequestPage params={Promise.resolve({ owner: "acme", repo: "demo" })} />);

    await waitFor(() => {
      expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
    });

    await user.type(screen.getByLabelText(/base branch/i), "main");
    await user.type(screen.getByLabelText(/head branch/i), "feature");
    await user.click(screen.getByRole("button", { name: /create pull request/i }));

    expect(await screen.findByText("Title is required")).toBeInTheDocument();

    const pullPosts = fetchMock.mock.calls.filter(
      ([url, init]) =>
        String(url).includes("/api/v3/repos/acme/demo/pulls") && init?.method === "POST",
    );
    expect(pullPosts).toHaveLength(0);
  });

  it("TestPullsNewValidatesSameBaseHead", async () => {
    const user = userEvent.setup();

    render(<NewPullRequestPage params={Promise.resolve({ owner: "acme", repo: "demo" })} />);

    await waitFor(() => {
      expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
    });

    await user.type(screen.getByLabelText(/title/i), "My PR");
    await user.type(screen.getByLabelText(/base branch/i), "main");
    await user.type(screen.getByLabelText(/head branch/i), "main");
    await user.click(screen.getByRole("button", { name: /create pull request/i }));

    expect(
      await screen.findByText("Head branch must differ from base branch"),
    ).toBeInTheDocument();

    const pullPosts = fetchMock.mock.calls.filter(
      ([url, init]) =>
        String(url).includes("/api/v3/repos/acme/demo/pulls") && init?.method === "POST",
    );
    expect(pullPosts).toHaveLength(0);
  });

  it("TestPullsNewSubmitSuccess", async () => {
    const user = userEvent.setup();

    render(<NewPullRequestPage params={Promise.resolve({ owner: "acme", repo: "demo" })} />);

    await waitFor(() => {
      expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
    });

    await user.type(screen.getByLabelText(/title/i), "Add feature");
    await user.type(screen.getByLabelText(/base branch/i), "main");
    await user.type(screen.getByLabelText(/head branch/i), "feature");
    await user.type(screen.getByLabelText(/body/i), "Summary of changes");
    await user.click(screen.getByRole("button", { name: /create pull request/i }));

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith("/acme/demo/pull/7");
    });

    const pullPosts = fetchMock.mock.calls.filter(
      ([url, init]) =>
        String(url).includes("/api/v3/repos/acme/demo/pulls") && init?.method === "POST",
    );
    expect(pullPosts).toHaveLength(1);
    expect(JSON.parse(String(pullPosts[0][1]?.body))).toEqual({
      title: "Add feature",
      body: "Summary of changes",
      head: "feature",
      base: "main",
    });
  });

  it("TestPullsNewSubmitError422", async () => {
    fetchMock.mockImplementation((url: string | URL, init?: RequestInit) => {
      const urlStr = String(url);

      if (init?.method === "POST" && urlStr.includes("/api/v3/repos/acme/demo/pulls")) {
        return Promise.resolve({
          ok: false,
          status: 422,
          statusText: "Unprocessable Entity",
          json: async () => ({ message: "Branch not found" }),
        });
      }

      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({}),
      });
    });

    const user = userEvent.setup();

    render(<NewPullRequestPage params={Promise.resolve({ owner: "acme", repo: "demo" })} />);

    await waitFor(() => {
      expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
    });

    await user.type(screen.getByLabelText(/title/i), "Valid title");
    await user.type(screen.getByLabelText(/base branch/i), "main");
    await user.type(screen.getByLabelText(/head branch/i), "missing-branch");
    await user.click(screen.getByRole("button", { name: /create pull request/i }));

    expect(await screen.findByText("Branch not found")).toBeInTheDocument();
    expect(mockPush).not.toHaveBeenCalled();
  });
});
