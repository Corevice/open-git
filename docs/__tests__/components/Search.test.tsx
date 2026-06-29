import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import Search from "../../components/Search";

describe("Search", () => {
  it("renders disabled search input when pagefind index is unavailable", () => {
    delete (window as { pagefind?: unknown }).pagefind;

    render(<Search />);

    const input = screen.getByRole("searchbox", { name: "ドキュメント検索" });
    expect(input).toBeDisabled();
    expect(input).toHaveAttribute("aria-disabled", "true");
  });
});
