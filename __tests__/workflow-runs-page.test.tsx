import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import ActionsListPage from "@/app/10-actions-list/page";

const mockPush = vi.fn();
const fetchMock = vi.fn();

const mockRunsResponse = {
  workflow_runs: [
    {
      id: 101,
      name: "CI Build",
      run_number: 42,
      status: "completed",
      conclusion: "success",
      head_branch: "main",
      head_sha: "abc123def456",
      event: "push",
      started_at: "2026-06-28T10:00:00Z",
      completed_at: "2026-06-28T10:02:14Z",
      created_at: "2026-06-28T09:59:00Z",
    },
    {
      id: 102,
      name: "Deploy",
      run_number: 43,
      status: "in_progress",
      conclusion: null,
      head_branch: "develop",
      head_sha: "fedcba987654",
      event: "workflow_dispatch",
      started_at: "2026-06-28T11:00:00Z",
      created_at: "2026-06-28T10:59:00Z",
    },
  ],
  total_count: 2,
};

vi.mock("next/navigation", () => ({
  useSearchParams: () => new URLSearchParams("owner=alice&repo=hello"),
  useRouter: () => ({ push: mockPush }),
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => ({
    token: "test-token",
    isAuthenticated: true,
    login: vi.fn(),
    logout: vi.fn(),
  }),
}));

describe("ActionsListPage", () => {
  beforeEach(() => {
    mockPush.mockClear();
    fetchMock.mockReset();
    vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "http://localhost:8080");
    vi.stubGlobal("fetch", fetchMock);
  });

  it("renders run rows from mocked API response", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => mockRunsResponse,
    });

    render(<ActionsListPage />);

    await waitFor(() => {
      expect(screen.getByText("CI Build")).toBeInTheDocument();
    });

    expect(screen.getByText("#42")).toBeInTheDocument();
    expect(screen.getByText("Deploy")).toBeInTheDocument();
    expect(screen.getByText("#43")).toBeInTheDocument();
    expect(screen.getByText("main")).toBeInTheDocument();
    expect(screen.getByText("abc123d")).toBeInTheDocument();

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/repos/alice/hello/actions/runs?status=&branch=",
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: "Bearer test-token",
        }),
      }),
    );
  });

  it("shows skeleton rows while loading", () => {
    fetchMock.mockReturnValue(new Promise(() => {}));

    render(<ActionsListPage />);

    expect(screen.getAllByTestId("run-skeleton-row")).toHaveLength(5);
  });

  it("refetches when status filter changes", async () => {
    const user = userEvent.setup();
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => mockRunsResponse,
    });

    render(<ActionsListPage />);

    await waitFor(() => {
      expect(screen.getByText("CI Build")).toBeInTheDocument();
    });

    fetchMock.mockClear();
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ workflow_runs: [], total_count: 0 }),
    });

    await user.selectOptions(screen.getByLabelText("Status filter"), "completed");

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "http://localhost:8080/repos/alice/hello/actions/runs?status=completed&branch=",
        expect.any(Object),
      );
    });
  });
});
