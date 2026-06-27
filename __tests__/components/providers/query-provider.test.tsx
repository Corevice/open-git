import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

import { QueryProvider } from "@/components/providers/query-provider";

describe("QueryProvider", () => {
  it("renders children without throwing", () => {
    expect(() =>
      render(
        <QueryProvider>
          <span>ok</span>
        </QueryProvider>,
      ),
    ).not.toThrow();

    expect(screen.getByText("ok")).toBeVisible();
  });
});
