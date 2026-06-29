import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import ContributorList from "@/components/docs/ContributorList";

describe("ContributorList", () => {
  it("renders contributor login", () => {
    render(
      <ContributorList
        contributors={[
          {
            login: "alice",
            id: 1,
            avatar_url: "https://example.com/alice.png",
            contributions: 5,
            type: "User",
          },
        ]}
      />,
    );

    expect(screen.getByText("alice")).toBeInTheDocument();
  });

  it("renders initials when avatar_url is empty", () => {
    render(
      <ContributorList
        contributors={[
          {
            login: "alice",
            id: 1,
            avatar_url: "",
            contributions: 5,
            type: "User",
          },
        ]}
      />,
    );

    expect(screen.getByText("AL")).toBeInTheDocument();
  });
});
