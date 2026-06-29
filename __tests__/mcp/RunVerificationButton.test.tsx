import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RunVerificationButton } from "@/components/mcp/RunVerificationButton";

const { runMCPVerification, getMCPJobStatus } = vi.hoisted(() => ({
  runMCPVerification: vi.fn(),
  getMCPJobStatus: vi.fn(),
}));

vi.mock("@/lib/api-client", () => ({
  apiClient: {
    runMCPVerification,
    getMCPJobStatus,
  },
  isApiError: vi.fn(() => false),
}));

describe("RunVerificationButton", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("TestIdleState: button is enabled, no spinner present", () => {
    const { container } = render(<RunVerificationButton />);

    expect(
      screen.getByRole("button", { name: /run verification/i }),
    ).not.toBeDisabled();
    expect(container.querySelector(".animate-spin")).toBeNull();
  });

  it("TestLoadingState: button is disabled after submit while job is running", async () => {
    runMCPVerification.mockResolvedValue({
      job_id: "j1",
      status: "queued",
    });
    getMCPJobStatus.mockResolvedValue({
      job_id: "j1",
      status: "running",
      progress: 0.5,
    });

    render(<RunVerificationButton />);

    fireEvent.change(screen.getByPlaceholderText("owner/repo"), {
      target: { value: "octo-org/hello-repo" },
    });
    fireEvent.click(screen.getByRole("button", { name: /run verification/i }));

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /run verification/i }),
      ).toBeDisabled();
    });
  });

  it("TestErrorState: shows destructive Alert when runMCPVerification rejects", async () => {
    runMCPVerification.mockRejectedValue(new Error("network error"));

    render(<RunVerificationButton />);

    fireEvent.change(screen.getByPlaceholderText("owner/repo"), {
      target: { value: "octo-org/hello-repo" },
    });
    fireEvent.click(screen.getByRole("button", { name: /run verification/i }));

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveClass("border-destructive/50");
      expect(screen.getByText("network error")).toBeInTheDocument();
    });
  });
});
