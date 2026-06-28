import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

import { CompatBadge } from "@/components/ui/compat-badge";

describe("CompatBadge", () => {
  it("renders pass badge with green colour class", () => {
    render(<CompatBadge status="pass" />);

    const badge = screen.getByText("Pass");
    expect(badge).toHaveClass("bg-[#dafbe1]");
    expect(badge).toHaveClass("text-[#1a7f37]");
  });

  it("renders fail badge with red colour class", () => {
    render(<CompatBadge status="fail" />);

    const badge = screen.getByText("Fail");
    expect(badge).toHaveClass("bg-[#ffebe9]");
    expect(badge).toHaveClass("text-[#cf222e]");
  });

  it("renders untested badge", () => {
    render(<CompatBadge status="untested" />);

    const badge = screen.getByText("Untested");
    expect(badge).toHaveClass("bg-[#eaeef2]");
    expect(badge).toHaveClass("text-[#656d76]");
  });

  it("renders partial badge", () => {
    render(<CompatBadge status="partial" />);

    const badge = screen.getByText("Partial");
    expect(badge).toHaveClass("bg-[#fff8c5]");
    expect(badge).toHaveClass("text-[#9a6700]");
  });
});
