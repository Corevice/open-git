import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import ErrorBoundary from "@/components/ErrorBoundary";

function ThrowingChild(): never {
  throw new Error("Something went wrong");
}

describe("ErrorBoundary", () => {
  it("renders fallback UI with message and Retry button when child throws", () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

    render(
      <ErrorBoundary>
        <ThrowingChild />
      </ErrorBoundary>,
    );

    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();

    consoleError.mockRestore();
  });
});
