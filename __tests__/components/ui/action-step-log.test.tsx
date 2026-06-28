import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect } from "vitest";

import { ActionStepLog } from "@/components/ui/action-step-log";

describe("ActionStepLog", () => {
  it("renders step name in collapsed state", () => {
    render(
      <ActionStepLog
        stepName="Run tests"
        conclusion="success"
        log="All tests passed"
      />,
    );

    expect(screen.getByText("Run tests")).toBeInTheDocument();
    expect(screen.queryByText("All tests passed")).not.toBeInTheDocument();
  });

  it("expands log on click", () => {
    render(
      <ActionStepLog
        stepName="Run tests"
        conclusion="success"
        log="All tests passed"
      />,
    );

    fireEvent.click(screen.getByRole("button"));

    expect(screen.getByText("All tests passed")).toBeInTheDocument();
  });

  it("hides log when clicked again", () => {
    render(
      <ActionStepLog
        stepName="Run tests"
        conclusion="success"
        log="All tests passed"
      />,
    );

    const button = screen.getByRole("button");
    fireEvent.click(button);
    fireEvent.click(button);

    expect(screen.queryByText("All tests passed")).not.toBeInTheDocument();
  });
});
