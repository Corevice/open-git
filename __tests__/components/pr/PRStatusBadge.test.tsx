import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

import { PRStatusBadge } from "@/components/pr/PRStatusBadge";

describe("PRStatusBadge", () => {
  it("renders open state", () => {
    render(<PRStatusBadge state="open" />);
    expect(screen.getByText("Open")).toBeInTheDocument();
  });

  it("renders merged state", () => {
    render(<PRStatusBadge state="merged" />);
    expect(screen.getByText("Merged")).toBeInTheDocument();
  });

  it("renders closed state", () => {
    render(<PRStatusBadge state="closed" />);
    expect(screen.getByText("Closed")).toBeInTheDocument();
  });
});
