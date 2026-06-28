import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import Error from "@/app/(app)/[owner]/[repo]/error";

describe("Repo error boundary", () => {
  it("shows error message and calls reset on Retry click", async () => {
    const reset = vi.fn();
    const error = new Error("Failed to load repository");

    render(<Error error={error} reset={reset} />);

    expect(screen.getByText("Failed to load repository")).toBeInTheDocument();

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Retry" }));

    expect(reset).toHaveBeenCalledOnce();
  });
});
