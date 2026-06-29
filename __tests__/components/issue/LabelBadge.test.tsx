import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

import { LabelBadge } from "@/components/issue/LabelBadge";

describe("LabelBadge", () => {
  it("renders name and backgroundColor style", () => {
    const { container } = render(<LabelBadge name="bug" color="d73a4a" />);

    expect(screen.getByText("bug")).toBeInTheDocument();
    const badge = container.querySelector("span");
    expect(badge?.style.backgroundColor).toBeTruthy();
  });
});
