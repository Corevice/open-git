import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import ActionsRunDetailPage from "@/app/11-actions-run/page";

const mockPush = vi.fn();

const inProgressRun = {
  id: 42,
  name: "CI",
  run_number: 7,
  status: "in_progress",
  conclusion: null,
  head_branch: "main",
  head_sha: "abc1234567890",
  event: "push",
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:02:00Z",
  run_started_at: "2024-01-01T00:00:00Z",
  actor: { login: "octocat" },
};

const completedRun = {
  ...inProgressRun,
  status: "completed",
  conclusion: "success",
};

const mockJobs = {
  jobs: [
    {
      id: 101,
      name: "build",
      status: "completed",
      conclusion: "success",
      runner_label: "ubuntu-latest",
      started_at: "2024-01-01T00:00:10Z",
      completed_at: "2024-01-01T00:01:00Z",
    },
    {
      id: 102,
      name: "test",
      status: "in_progress",
      conclusion: null,
      runner_label: "ubuntu-latest",
      started_at: "2024-01-01T00:01:10Z",
      completed_at: null,
    },
  ],
};

function createFetchMock(run: typeof inProgressRun) {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = init?.method ?? "GET";

    if (method === "POST" && url.includes("/cancel")) {
      return {
        ok: true,
        status: 202,
        json: async () => ({}),
      };
    }

    if (method === "POST" && url.includes("/rerun")) {
      return {
        ok: true,
        status: 202,
        json: async () => ({}),
      };
    }

    if (url.includes("/actions/runs/42/jobs")) {
      return {
        ok: true,
        status: 200,
        json: async () => mockJobs,
      };
    }

    if (url.includes("/actions/runs/42")) {
      return {
        ok: true,
        status: 200,
        json: async () => run,
      };
    }

    throw new Error(`Unexpected fetch: ${method} ${url}`);
  });
}

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
  useSearchParams: () => new URLSearchParams("runId=42&owner=openhub&repo=awesome-project"),
  useParams: () => ({}),
}));

describe("ActionsRunDetailPage", () => {
  beforeEach(() => {
    mockPush.mockClear();
    vi.stubGlobal("fetch", createFetchMock(inProgressRun));
  });

  it("renders run summary with status badge, branch, and SHA", async () => {
    render(<ActionsRunDetailPage />);

    expect(await screen.findByText("CI")).toBeInTheDocument();
    expect(screen.getByText("#7")).toBeInTheDocument();
    expect(screen.getAllByText("In progress").length).toBeGreaterThan(0);
    expect(screen.getByText("main")).toBeInTheDocument();
    expect(screen.getByText("abc1234")).toBeInTheDocument();
  });

  it("renders both jobs from the jobs endpoint", async () => {
    render(<ActionsRunDetailPage />);

    expect(await screen.findByText("build")).toBeInTheDocument();
    expect(screen.getByText("test")).toBeInTheDocument();
    expect(screen.getAllByText("ubuntu-latest")).toHaveLength(2);
  });

  it("shows Cancel button for in_progress runs and calls cancel endpoint", async () => {
    const user = userEvent.setup();
    const fetchMock = createFetchMock(inProgressRun);
    vi.stubGlobal("fetch", fetchMock);

    render(<ActionsRunDetailPage />);

    const cancelButton = await screen.findByRole("button", { name: "Cancel" });
    expect(cancelButton).toBeInTheDocument();

    await user.click(cancelButton);

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/repos/openhub/awesome-project/actions/runs/42/cancel",
        expect.objectContaining({ method: "POST" }),
      );
    });
  });

  it("hides Cancel button for completed runs", async () => {
    vi.stubGlobal("fetch", createFetchMock(completedRun));

    render(<ActionsRunDetailPage />);

    expect(await screen.findByRole("button", { name: "Re-run" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Cancel" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Re-run" })).toBeInTheDocument();
  });
});
