import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { Sidebar } from "@/components/layout/Sidebar";

describe("Sidebar", () => {
  it("open=true renders drawer with nav links visible", () => {
    render(<Sidebar open={true} onClose={vi.fn()} />);

    const drawer = screen.getByTestId("sidebar-drawer");

    expect(
      within(drawer).getByRole("link", { name: "Dashboard" }),
    ).toBeInTheDocument();
    expect(
      within(drawer).getByRole("link", { name: "Repositories" }),
    ).toBeInTheDocument();
    expect(
      within(drawer).getByRole("link", { name: "Issues" }),
    ).toBeInTheDocument();
    expect(
      within(drawer).getByRole("link", { name: "Pull Requests" }),
    ).toBeInTheDocument();
  });

  it("open=false does not render drawer content", () => {
    render(<Sidebar open={false} onClose={vi.fn()} />);

    expect(
      screen.queryByTestId("sidebar-backdrop"),
    ).not.toBeInTheDocument();
  });

  it("clicking backdrop calls onClose", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();

    render(<Sidebar open={true} onClose={onClose} />);

    await user.click(screen.getByTestId("sidebar-backdrop"));

    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
