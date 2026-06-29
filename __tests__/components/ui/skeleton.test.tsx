import { render } from "@testing-library/react";
import { describe, it, expect } from "vitest";

import { SkeletonBlock } from "@/components/ui/skeleton";

describe("SkeletonBlock", () => {
  it("renders with animate-pulse class", () => {
    const { container } = render(<SkeletonBlock />);
    expect(container.firstChild).toHaveClass("animate-pulse");
  });
});
