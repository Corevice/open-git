import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { AdvisoryStatusForm } from "@/components/admin/AdvisoryStatusForm";

describe("AdvisoryStatusForm", () => {
  it("shows dismissed reason field when state is dismissed", async () => {
    const user = userEvent.setup();

    render(<AdvisoryStatusForm onSubmit={vi.fn()} />);

    expect(screen.queryByLabelText("Dismissed reason")).not.toBeInTheDocument();

    await user.selectOptions(screen.getByLabelText("State"), "dismissed");

    expect(screen.getByLabelText("Dismissed reason")).toBeInTheDocument();
  });

  it("hides dismissed reason field for non-dismissed states", async () => {
    const user = userEvent.setup();

    render(<AdvisoryStatusForm onSubmit={vi.fn()} />);

    await user.selectOptions(screen.getByLabelText("State"), "dismissed");
    expect(screen.getByLabelText("Dismissed reason")).toBeInTheDocument();

    await user.selectOptions(screen.getByLabelText("State"), "acknowledged");

    expect(screen.queryByLabelText("Dismissed reason")).not.toBeInTheDocument();
  });

  it("shows validation error when submitting dismissed without reason", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();

    render(<AdvisoryStatusForm onSubmit={onSubmit} />);

    await user.selectOptions(screen.getByLabelText("State"), "dismissed");
    await user.click(screen.getByRole("button", { name: "Update status" }));

    expect(
      screen.getByRole("alert"),
    ).toHaveTextContent("Dismissed reason is required when state is dismissed.");
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("calls onSubmit with state and reason on valid dismissed submit", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();

    render(<AdvisoryStatusForm onSubmit={onSubmit} />);

    await user.selectOptions(screen.getByLabelText("State"), "dismissed");
    await user.selectOptions(
      screen.getByLabelText("Dismissed reason"),
      "tolerable_risk",
    );
    await user.click(screen.getByRole("button", { name: "Update status" }));

    expect(onSubmit).toHaveBeenCalledTimes(1);
    expect(onSubmit).toHaveBeenCalledWith("dismissed", "tolerable_risk");
  });

  it("calls onSubmit with state only for non-dismissed submit", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();

    render(<AdvisoryStatusForm onSubmit={onSubmit} />);

    await user.selectOptions(screen.getByLabelText("State"), "resolved");
    await user.click(screen.getByRole("button", { name: "Update status" }));

    expect(onSubmit).toHaveBeenCalledTimes(1);
    expect(onSubmit).toHaveBeenCalledWith("resolved");
  });
});
