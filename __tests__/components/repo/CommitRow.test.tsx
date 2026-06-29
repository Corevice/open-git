import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

import CommitRow from "@/components/repo/CommitRow";
import type { CommitEntry } from "@/types/repo";

vi.mock("next/link", () => ({
  default: ({
    href,
    children,
    ...props
  }: {
    href: string;
    children: React.ReactNode;
  }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

describe("CommitRow", () => {
  it("truncates sha and commit message", () => {
    const entry: CommitEntry = {
      sha: "abcdef1234567",
      commit: {
        message:
          "A very long commit message that exceeds seventy-two characters in total length for sure",
        author: {
          name: "Alice",
          date: new Date().toISOString(),
        },
      },
    };

    render(<CommitRow entry={entry} owner="acme" repo="demo" />);

    expect(screen.getByText("abcdef1")).toBeInTheDocument();
    const message = screen.getByText(/A very long commit message/);
    expect(message.textContent?.endsWith("…")).toBe(true);
    expect(message.textContent!.length).toBeGreaterThan(72);
  });
});
