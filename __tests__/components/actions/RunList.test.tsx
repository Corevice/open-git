import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";

import { RunList, type WorkflowRun } from "@/components/actions/RunList";

const mockRuns: WorkflowRun[] = [
  {
    id: 1,
    name: "CI",
    run_number: 42,
    status: "completed",
    conclusion: "success",
    head_branch: "main",
    head_sha: "abc1234567890",
    event: "push",
    actor: { login: "octocat" },
    run_started_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:05:00Z",
  },
  {
    id: 2,
    name: "Deploy",
    run_number: 7,
    status: "queued",
    conclusion: null,
    head_branch: "develop",
    head_sha: "def9876543210",
    event: "workflow_dispatch",
    actor: { login: "hubot" },
  },
];

const defaultProps = {
  runs: mockRuns,
  loading: false,
  error: null,
  statusFilter: "all",
  onStatusFilterChange: vi.fn(),
  branchFilter: "",
  onBranchFilterChange: vi.fn(),
  eventFilter: "all",
  onEventFilterChange: vi.fn(),
  page: 1,
  totalCount: 2,
  perPage: 30,
  onPageChange: vi.fn(),
};

describe("RunList", () => {
  it("renders table rows for each run", () => {
    render(<RunList {...defaultProps} />);

    expect(screen.getByText("CI")).toBeInTheDocument();
    expect(screen.getByText("Deploy")).toBeInTheDocument();
    expect(screen.getByText("#42")).toBeInTheDocument();
    expect(screen.getByText("#7")).toBeInTheDocument();
  });

  it("renders filter controls", () => {
    render(<RunList {...defaultProps} />);

    expect(screen.getByLabelText("Filter by status")).toBeInTheDocument();
    expect(screen.getByLabelText("Filter by branch")).toBeInTheDocument();
    expect(screen.getByLabelText("Filter by event")).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "queued" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "push" })).toBeInTheDocument();
  });

  it("disables Previous button on page 1", () => {
    render(<RunList {...defaultProps} page={1} totalCount={60} />);

    expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Next" })).not.toBeDisabled();
  });
});
