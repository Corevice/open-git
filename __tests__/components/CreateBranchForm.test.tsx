import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import CreateBranchForm from "@/components/repo/CreateBranchForm";
import { apiClient } from "@/lib/api-client";

const mockRefresh = vi.fn();

vi.mock("@/lib/api-client", () => ({
  apiClient: {
    createRef: vi.fn(),
  },
  isApiError: (err: unknown) =>
    typeof err === "object" &&
    err !== null &&
    "status" in err &&
    "message" in err,
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ refresh: mockRefresh }),
}));

describe("CreateBranchForm", () => {
  const branches = [
    { name: "main", commit: { sha: "abc1234567890abcdef" } },
    { name: "develop", commit: { sha: "def4567890abcdef12" } },
  ];

  beforeEach(() => {
    mockRefresh.mockClear();
    vi.mocked(apiClient.createRef).mockReset();
    vi.mocked(apiClient.createRef).mockResolvedValue(undefined);
  });

  it("rejects empty branch name", async () => {
    const user = userEvent.setup();

    render(<CreateBranchForm owner="acme" repo="demo" branches={branches} />);

    await user.click(screen.getByRole("button", { name: /create branch/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Branch name is required",
    );
    expect(apiClient.createRef).not.toHaveBeenCalled();
  });

  it("calls createRef with refs/heads/ prefix and correct owner/repo", async () => {
    const user = userEvent.setup();

    render(<CreateBranchForm owner="acme" repo="demo" branches={branches} />);

    await user.type(screen.getByLabelText(/branch name/i), "feature-x");
    await user.click(screen.getByRole("button", { name: /create branch/i }));

    await waitFor(() => {
      expect(apiClient.createRef).toHaveBeenCalledWith(
        "acme",
        "demo",
        "refs/heads/feature-x",
        "abc1234567890abcdef",
      );
      expect(mockRefresh).toHaveBeenCalled();
    });
  });
});
