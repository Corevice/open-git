import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";

import { RunActions } from "@/components/actions/RunActions";

vi.mock("@/lib/api/actions", () => ({
  rerunRun: vi.fn(),
  rerunFailedJobs: vi.fn(),
  cancelRun: vi.fn(),
}));

const defaultProps = {
  owner: "acme",
  repo: "demo",
  runId: "123",
  conclusion: null as string | null,
  userCanWrite: true,
  onActionComplete: vi.fn(),
};

describe("RunActions", () => {
  it("shows Cancel button when status is in_progress", () => {
    render(<RunActions {...defaultProps} status="in_progress" />);

    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
  });

  it("does not show Re-run button when status is in_progress", () => {
    render(<RunActions {...defaultProps} status="in_progress" />);

    expect(
      screen.queryByRole("button", { name: "Re-run all jobs" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Re-run failed jobs" }),
    ).not.toBeInTheDocument();
  });

  it("disables all buttons when userCanWrite is false", () => {
    render(
      <RunActions
        {...defaultProps}
        status="completed"
        conclusion="failure"
        userCanWrite={false}
      />,
    );

    expect(screen.getByRole("button", { name: "Re-run all jobs" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Re-run failed jobs" })).toBeDisabled();
  });
});
