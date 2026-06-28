import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/hooks/useJobLogStream", () => ({
  useJobLogStream: vi.fn(),
}));

import { JobLogViewer } from "@/components/actions/JobLogViewer";
import { useJobLogStream } from "@/hooks/useJobLogStream";

const mockUseJobLogStream = vi.mocked(useJobLogStream);

const defaultProps = {
  owner: "octocat",
  repo: "hello-world",
  jobId: "123",
  isActive: true,
};

describe("JobLogViewer", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("strips ANSI escape codes from displayed text", () => {
    mockUseJobLogStream.mockReturnValue({
      lines: ["\x1b[32mHello\x1b[0m"],
      streaming: false,
    });

    render(<JobLogViewer {...defaultProps} />);

    expect(screen.getByText("Hello")).toBeInTheDocument();
  });

  it("renders at most 2000 line elements when lines exceed 2000", () => {
    const lines = Array.from({ length: 2001 }, (_, index) => `line ${index}`);
    mockUseJobLogStream.mockReturnValue({
      lines,
      streaming: false,
    });

    const { container } = render(<JobLogViewer {...defaultProps} />);

    expect(container.querySelectorAll(".whitespace-pre-wrap")).toHaveLength(2000);
  });
});
