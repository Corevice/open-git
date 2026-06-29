import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

import { EmptyState } from "@/components/ui/empty-state";

describe("EmptyState", () => {
  it("renders title in heading", () => {
    render(<EmptyState title="No results" description="Try again" />);
    expect(
      screen.getByRole("heading", { name: "No results" }),
    ).toBeInTheDocument();
  });

  it("renders action link when action prop given", () => {
    render(
      <EmptyState
        title="No repos"
        description="Create one"
        action={{ label: "New repo", href: "/new" }}
      />,
    );
    expect(screen.getByRole("link", { name: "New repo" })).toHaveAttribute(
      "href",
      "/new",
    );
  });

  it("does NOT render action when prop omitted", () => {
    render(<EmptyState title="Empty" description="Nothing here" />);
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });
});
