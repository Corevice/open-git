import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { Pagination } from "@/components/ui/pagination";

vi.mock("next/link", () => ({
  default: ({
    href,
    children,
    ...props
  }: {
    href: string;
    children: ReactNode;
  }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

describe("Pagination", () => {
  it("sets Previous aria-disabled when hasPrev is false", () => {
    render(
      <Pagination
        page={1}
        hasPrev={false}
        hasNext
        basePath="/owner/repo/commits"
      />,
    );

    expect(screen.getByRole("link", { name: "Previous" })).toHaveAttribute(
      "aria-disabled",
      "true",
    );
  });

  it("sets Next aria-disabled when hasNext is false", () => {
    render(
      <Pagination
        page={3}
        hasPrev
        hasNext={false}
        basePath="/owner/repo/commits"
      />,
    );

    expect(screen.getByRole("link", { name: "Next" })).toHaveAttribute(
      "aria-disabled",
      "true",
    );
  });

  it("builds correct page query URLs", () => {
    render(
      <Pagination
        page={2}
        hasPrev
        hasNext
        basePath="/owner/repo/commits"
      />,
    );

    expect(screen.getByRole("link", { name: "Previous" })).toHaveAttribute(
      "href",
      "/owner/repo/commits?page=1",
    );
    expect(screen.getByRole("link", { name: "Next" })).toHaveAttribute(
      "href",
      "/owner/repo/commits?page=3",
    );
  });
});
