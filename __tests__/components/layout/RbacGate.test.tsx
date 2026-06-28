import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { RbacGate } from "@/components/layout/RbacGate";

describe("RbacGate", () => {
  it('requiredRole="admin", userRole="write" — children not rendered', () => {
    render(
      <RbacGate requiredRole="admin" userRole="write">
        <button type="button">Delete repository</button>
      </RbacGate>,
    );

    expect(
      screen.queryByRole("button", { name: "Delete repository" }),
    ).not.toBeInTheDocument();
  });

  it('requiredRole="write", userRole="admin" — children rendered', () => {
    render(
      <RbacGate requiredRole="write" userRole="admin">
        <button type="button">Edit issue</button>
      </RbacGate>,
    );

    expect(
      screen.getByRole("button", { name: "Edit issue" }),
    ).toBeInTheDocument();
  });

  it('requiredRole="read", userRole="read" — children rendered', () => {
    render(
      <RbacGate requiredRole="read" userRole="read">
        <span>View content</span>
      </RbacGate>,
    );

    expect(screen.getByText("View content")).toBeInTheDocument();
  });

  it('requiredRole="admin", userRole=null — children not rendered', () => {
    render(
      <RbacGate requiredRole="admin" userRole={null}>
        <button type="button">Delete repository</button>
      </RbacGate>,
    );

    expect(
      screen.queryByRole("button", { name: "Delete repository" }),
    ).not.toBeInTheDocument();
  });

  it('requiredRole="admin", userRole=undefined — children not rendered', () => {
    render(
      <RbacGate requiredRole="admin" userRole={undefined}>
        <button type="button">Delete repository</button>
      </RbacGate>,
    );

    expect(
      screen.queryByRole("button", { name: "Delete repository" }),
    ).not.toBeInTheDocument();
  });
});
