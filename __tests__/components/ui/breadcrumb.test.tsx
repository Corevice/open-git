import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

import { Breadcrumb } from "@/components/ui/breadcrumb";

describe("Breadcrumb", () => {
  const items = [
    { label: "Home", href: "/" },
    { label: "Repos", href: "/repos" },
    { label: "open-git" },
  ];

  it('has aria-label="Breadcrumb"', () => {
    render(<Breadcrumb items={items} />);
    expect(
      screen.getByRole("navigation", { name: "Breadcrumb" }),
    ).toBeInTheDocument();
  });

  it("last item has no anchor", () => {
    render(<Breadcrumb items={items} />);
    const lastItem = screen.getByText("open-git");
    expect(lastItem.tagName).toBe("SPAN");
    expect(lastItem.closest("a")).toBeNull();
  });

  it("first item has correct href", () => {
    render(<Breadcrumb items={items} />);
    expect(screen.getByRole("link", { name: "Home" })).toHaveAttribute(
      "href",
      "/",
    );
  });
});
