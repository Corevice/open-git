import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

import LogStreamStatus from "@/components/actions/LogStreamStatus";

describe("LogStreamStatus", () => {
  it.each([
    ["streaming", "Streaming"],
    ["completed", "Completed"],
    ["reconnecting", "Reconnecting"],
    ["failed", "Failed"],
  ] as const)("renders aria-label %s for status %s", (status, label) => {
    render(<LogStreamStatus status={status} />);

    expect(screen.getByRole("status", { name: label })).toBeInTheDocument();
  });
});
