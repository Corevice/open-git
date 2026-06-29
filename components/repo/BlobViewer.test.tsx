import { render, screen } from "@testing-library/react";
import BlobViewer from "./BlobViewer";

vi.mock("prismjs", () => ({
  default: {
    highlight: (s: string) => s,
    highlightAll: () => {},
    languages: { plaintext: {} },
  },
}));

vi.mock("prismjs/components/prism-bash", () => ({}));
vi.mock("prismjs/components/prism-css", () => ({}));
vi.mock("prismjs/components/prism-go", () => ({}));
vi.mock("prismjs/components/prism-javascript", () => ({}));
vi.mock("prismjs/components/prism-json", () => ({}));
vi.mock("prismjs/components/prism-markdown", () => ({}));
vi.mock("prismjs/components/prism-python", () => ({}));
vi.mock("prismjs/components/prism-sql", () => ({}));
vi.mock("prismjs/components/prism-tsx", () => ({}));
vi.mock("prismjs/components/prism-typescript", () => ({}));
vi.mock("prismjs/components/prism-yaml", () => ({}));
vi.mock("prismjs/themes/prism.css", () => ({}));

describe("BlobViewer", () => {
  it("renders binary notice when binary=true", () => {
    render(<BlobViewer content="" filename="img.png" binary={true} />);

    expect(screen.getByText(/Binary file/i)).toBeInTheDocument();
  });

  it("shows truncation banner when truncated=true", () => {
    render(
      <BlobViewer
        content=""
        filename="large.txt"
        truncated={true}
        rawUrl="http://x"
      />,
    );

    expect(screen.getByText(/File is too large to display/i)).toBeInTheDocument();
  });

  it("rawUrl link appears in truncated banner", () => {
    render(
      <BlobViewer
        content=""
        filename="large.txt"
        truncated={true}
        rawUrl="http://x"
      />,
    );

    const link = screen.getByRole("link", { name: /View raw/i });
    expect(link).toHaveAttribute("href", "http://x");
  });

  it("renders correct number of line rows", () => {
    const { container } = render(
      <BlobViewer
        content={"line1\nline2\nline3"}
        filename="sample.txt"
        binary={false}
        truncated={false}
      />,
    );

    const rows = container.querySelectorAll("tbody tr");
    expect(rows).toHaveLength(3);
  });
});
