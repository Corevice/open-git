import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import StatusBadge from "@/components/StatusBadge";

describe("StatusBadge", () => {
  it("renders queued label", () => {
    render(<StatusBadge status="queued" />);
    expect(screen.getByText("Queued")).toBeInTheDocument();
  });

  it("renders in_progress label with animate-pulse", () => {
    render(<StatusBadge status="in_progress" />);
    const badge = screen.getByText("In progress");
    expect(badge).toBeInTheDocument();
    expect(badge.className).toContain("animate-pulse");
    expect(badge.className).toContain("bg-yellow-400");
  });

  it("renders completed+success label", () => {
    render(<StatusBadge status="completed" conclusion="success" />);
    const badge = screen.getByText("Success");
    expect(badge).toBeInTheDocument();
    expect(badge.className).toContain("bg-green-500");
  });

  it("renders completed+failure label", () => {
    render(<StatusBadge status="completed" conclusion="failure" />);
    const badge = screen.getByText("Failure");
    expect(badge).toBeInTheDocument();
    expect(badge.className).toContain("bg-red-500");
  });

  it("renders completed+cancelled label", () => {
    render(<StatusBadge status="completed" conclusion="cancelled" />);
    const badge = screen.getByText("Cancelled");
    expect(badge).toBeInTheDocument();
    expect(badge.className).toContain("bg-gray-400");
  });

  it("renders completed+skipped label", () => {
    render(<StatusBadge status="completed" conclusion="skipped" />);
    const badge = screen.getByText("Skipped");
    expect(badge).toBeInTheDocument();
    expect(badge.className).toContain("bg-gray-300");
  });
});
