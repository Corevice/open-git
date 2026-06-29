import { render, screen } from "@testing-library/react";
import DiffViewer from "./DiffViewer";

describe("DiffViewer", () => {
  it("renders filename", () => {
    render(<DiffViewer filename="main.go" />);

    expect(screen.getByText("main.go")).toBeInTheDocument();
  });

  it("shows No diff when no patch", () => {
    render(<DiffViewer filename="main.go" />);

    expect(screen.getByText(/No diff available/i)).toBeInTheDocument();
  });

  it("renders hunk header", () => {
    render(
      <DiffViewer
        filename="main.go"
        patch="@@ -1,3 +1,4 @@ func\n line1\n+added"
      />,
    );

    expect(screen.getByText(/@@/)).toBeInTheDocument();
  });

  it("shows collapse button for large hunk", () => {
    const addedLines = Array.from({ length: 120 }, (_, i) => `+line${i}`).join(
      "\n",
    );
    const patch = `@@ -1,1 +1,120 @@\n${addedLines}`;

    render(<DiffViewer filename="big.go" patch={patch} />);

    expect(screen.getByText(/Show/i)).toBeInTheDocument();
  });

  it("addition line count shown in header", () => {
    render(<DiffViewer filename="main.go" additions={5} deletions={2} />);

    expect(screen.getByText("+5")).toBeInTheDocument();
  });
});
