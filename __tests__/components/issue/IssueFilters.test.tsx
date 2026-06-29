import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import IssueFilters from "@/components/issue/IssueFilters";

const mockPush = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
}));

describe("IssueFilters", () => {
  beforeEach(() => {
    mockPush.mockClear();
    vi.stubGlobal(
      "fetch",
      vi.fn((url: string) => {
        if (url.includes("/labels")) {
          return Promise.resolve({
            ok: true,
            json: async () => [
              { name: "bug", color: "d73a4a" },
              { name: "enhancement", color: "a2eeef" },
            ],
          });
        }
        if (url.includes("/milestones")) {
          return Promise.resolve({
            ok: true,
            json: async () => [{ number: 1, title: "v1.0", open_issues: 3 }],
          });
        }
        return Promise.resolve({ ok: true, json: async () => [] });
      }),
    );
  });

  it("renders Open/Closed toggle buttons", () => {
    render(
      <IssueFilters
        owner="octocat"
        repo="hello-world"
        state="open"
        basePath="/octocat/hello-world/issues"
      />,
    );

    expect(screen.getByRole("link", { name: "Open" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Closed" })).toBeInTheDocument();
  });

  it('clicking "Closed" navigates to ?state=closed&page=1', () => {
    render(
      <IssueFilters
        owner="octocat"
        repo="hello-world"
        state="open"
        basePath="/octocat/hello-world/issues"
      />,
    );

    const closedLink = screen.getByRole("link", { name: "Closed" });
    expect(closedLink).toHaveAttribute(
      "href",
      "/octocat/hello-world/issues?state=closed&page=1",
    );
  });

  it("label dropdown populates from fetched labels", async () => {
    render(
      <IssueFilters
        owner="octocat"
        repo="hello-world"
        state="open"
        basePath="/octocat/hello-world/issues"
      />,
    );

    await waitFor(() => {
      expect(screen.getByRole("option", { name: "bug" })).toBeInTheDocument();
    });

    expect(screen.getByRole("option", { name: "enhancement" })).toBeInTheDocument();
  });

  it("selecting a label navigates with updated query params", async () => {
    const user = userEvent.setup();

    render(
      <IssueFilters
        owner="octocat"
        repo="hello-world"
        state="open"
        basePath="/octocat/hello-world/issues"
      />,
    );

    await waitFor(() => {
      expect(screen.getByRole("option", { name: "bug" })).toBeInTheDocument();
    });

    await user.selectOptions(screen.getByLabelText("Filter by label"), "bug");

    expect(mockPush).toHaveBeenCalledWith("/octocat/hello-world/issues?labels=bug&page=1");
  });
});
