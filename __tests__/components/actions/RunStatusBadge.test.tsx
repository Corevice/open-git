import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

import { RunStatusBadge } from "@/components/actions/RunStatusBadge";

describe("RunStatusBadge", () => {
  it("renders queued status with yellow class", () => {
    const { container } = render(<RunStatusBadge status="queued" conclusion={null} />);

    expect(screen.getByText("Queued")).toBeInTheDocument();
    expect(container.querySelector("span")?.className).toContain("bg-[#fff8c5]");
    expect(container.querySelector("span")?.className).toContain("text-[#9a6700]");
  });

  it("renders failure conclusion with red class", () => {
    const { container } = render(
      <RunStatusBadge status="completed" conclusion="failure" />,
    );

    expect(screen.getByText("Failure")).toBeInTheDocument();
    expect(container.querySelector("span")?.className).toContain("bg-[#ffebe9]");
    expect(container.querySelector("span")?.className).toContain("text-[#cf222e]");
  });

  it("renders success conclusion with green class", () => {
    const { container } = render(
      <RunStatusBadge status="completed" conclusion="success" />,
    );

    expect(screen.getByText("Success")).toBeInTheDocument();
    expect(container.querySelector("span")?.className).toContain("bg-[#dafbe1]");
    expect(container.querySelector("span")?.className).toContain("text-[#1a7f37]");
  });
});
