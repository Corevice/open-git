import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import EmptyRepoState from "@/components/repo/EmptyRepoState";

describe("EmptyRepoState", () => {
  it('renders "No code yet" heading and clone URL', () => {
    const cloneUrl = "https://git.example.com/org/repo.git";

    render(<EmptyRepoState cloneUrl={cloneUrl} />);

    expect(
      screen.getByRole("heading", { name: "No code yet" }),
    ).toBeInTheDocument();
    expect(screen.getByText(cloneUrl)).toBeInTheDocument();
  });
});
