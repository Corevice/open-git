import { render, screen } from "@testing-library/react";
import FileTree from "./FileTree";

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

const defaultProps = {
  owner: "acme",
  repo: "demo",
  branch: "main",
  currentPath: "",
};

describe("FileTree", () => {
  it("dirs are sorted before files", () => {
    const { container } = render(
      <FileTree
        {...defaultProps}
        entries={[
          { type: "file", name: "b.txt", path: "b.txt" },
          { type: "dir", name: "a/", path: "a" },
        ]}
      />,
    );

    const firstLink = container.querySelector("a");
    expect(firstLink?.getAttribute("href")).toContain("/tree/");
  });

  it("empty entries renders empty state", () => {
    render(<FileTree {...defaultProps} entries={[]} />);

    expect(screen.getByText(/empty/i)).toBeInTheDocument();
  });

  it("file links contain /blob/", () => {
    const { container } = render(
      <FileTree
        {...defaultProps}
        entries={[{ type: "file", name: "readme.md", path: "readme.md" }]}
      />,
    );

    const link = container.querySelector("a");
    expect(link?.getAttribute("href")).toContain("/blob/");
  });

  it("dir links contain /tree/", () => {
    const { container } = render(
      <FileTree
        {...defaultProps}
        entries={[{ type: "dir", name: "src/", path: "src" }]}
      />,
    );

    const link = container.querySelector("a");
    expect(link?.getAttribute("href")).toContain("/tree/");
  });

  it("commit_message is displayed in row", () => {
    render(
      <FileTree
        {...defaultProps}
        entries={[
          {
            type: "file",
            name: "README.md",
            path: "README.md",
            commit_message: "Add readme",
          },
        ]}
      />,
    );

    expect(screen.getByText("Add readme")).toBeInTheDocument();
  });
});
