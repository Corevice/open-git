import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import FileTree from "@/components/repo/FileTree";

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

describe("FileTree", () => {
  it("fetches directory children on click and renders them", async () => {
    const apiClient = {
      getContents: vi.fn().mockResolvedValue([
        {
          name: "child.ts",
          path: "src/child.ts",
          type: "file",
          sha: "abc",
          size: 100,
        },
      ]),
    };

    render(
      <FileTree
        entries={[{ name: "src", path: "src", type: "dir" }]}
        owner="acme"
        repo="demo"
        treeRef="main"
        apiClient={apiClient}
      />,
    );

    await userEvent.click(screen.getByText("src"));

    expect(apiClient.getContents).toHaveBeenCalledWith(
      "acme",
      "demo",
      "src",
      "main",
    );

    await waitFor(() => {
      expect(screen.getByText("child.ts")).toBeInTheDocument();
    });
  });
});
