import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

import CommitList from "@/components/repo/CommitList";
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

function makeCommit(sha: string): CommitEntry {
  return {
    sha,
    commit: {
      message: `Commit ${sha}`,
      author: {
        name: "Alice",
        date: new Date().toISOString(),
      },
    },
  };
}

describe("CommitList", () => {
  it("renders empty state when there are no commits", () => {
    render(<CommitList commits={[]} owner="acme" repo="demo" />);

    expect(screen.getByText("No commits yet.")).toBeInTheDocument();
  });

  it("renders one list item per commit", () => {
    const commits = [makeCommit("aaa1111"), makeCommit("bbb2222"), makeCommit("ccc3333")];

    const { container } = render(
      <CommitList commits={commits} owner="acme" repo="demo" />,
    );

    expect(container.querySelectorAll("li")).toHaveLength(3);
  });
});
