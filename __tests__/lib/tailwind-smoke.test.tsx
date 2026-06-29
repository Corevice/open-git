import { render } from "@testing-library/react";
import { describe, it, expect } from "vitest";

describe("tailwind smoke", () => {
  it("renders Tailwind utility classes without throwing", () => {
    const { container } = render(
      <div className="bg-background text-foreground p-4">ok</div>,
    );
    expect(container).toBeTruthy();
  });
});
