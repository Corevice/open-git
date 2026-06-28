import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

import { AdminStatusCard } from "@/components/AdminStatusCard";

describe("AdminStatusCard", () => {
  it("renders ok status with green badge", () => {
    render(<AdminStatusCard name="Database" status="ok" />);

    expect(screen.getByText("ok").className).toContain("bg-green-500");
  });

  it("renders error status with red badge", () => {
    render(<AdminStatusCard name="Queue" status="error" />);

    expect(screen.getByText("error").className).toContain("bg-red-500");
  });

  it("renders unknown status with grey badge", () => {
    render(<AdminStatusCard name="Storage" status="unknown" />);

    expect(screen.getByText("unknown").className).toContain("bg-gray-400");
  });

  it("renders name prop in the DOM", () => {
    render(<AdminStatusCard name="Database" status="ok" />);

    expect(screen.getByText("Database")).toBeInTheDocument();
  });
});
