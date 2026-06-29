import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("prismjs", () => ({
  default: {
    highlightAllUnder: vi.fn(),
  },
}));

import MarkdownRenderer from "@/components/docs/MarkdownRenderer";

describe("MarkdownRenderer", () => {
  it("renders markdown content in a container div with HTML", () => {
    const content = "## Hello World\n\n```js\nconsole.log(1)\n```";
    const { container } = render(<MarkdownRenderer content={content} />);

    const div = container.querySelector(".prose");
    expect(div).toBeInTheDocument();
    expect(div?.innerHTML).not.toBe("");
  });
});
