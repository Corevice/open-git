import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import IssueList from "@/components/issue/IssueList";

vi.mock("next/navigation", () => ({
  useSearchParams: () => new URLSearchParams(),
}));

describe("IssueList empty state", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        headers: { get: () => null },
        json: async () => [],
      }),
    );
  });

  it('renders EmptyState with "No issues" text', async () => {
    render(
      <IssueList owner="octocat" repo="hello-world" state="open" page="1" />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("empty-state")).toBeInTheDocument();
    });

    expect(screen.getByText("No issues")).toBeInTheDocument();
  });
});
