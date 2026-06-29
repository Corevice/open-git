import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";

const mockSetTheme = vi.fn();

vi.mock("next-themes", () => ({
  useTheme: vi.fn(() => ({
    theme: "light",
    setTheme: mockSetTheme,
    resolvedTheme: "light",
  })),
}));

import { useTheme } from "next-themes";
import { ThemeToggle } from "@/components/ui/theme-toggle";

describe("ThemeToggle", () => {
  beforeEach(() => {
    mockSetTheme.mockClear();
    vi.mocked(useTheme).mockReturnValue({
      theme: "light",
      setTheme: mockSetTheme,
      resolvedTheme: "light",
      themes: ["light", "dark"],
    });
  });

  it("calls setTheme when clicked", async () => {
    const user = userEvent.setup();
    render(<ThemeToggle />);

    await user.click(screen.getByRole("button", { name: "Toggle theme" }));

    expect(mockSetTheme).toHaveBeenCalledWith("dark");
  });

  it("renders Moon icon when resolved theme is dark", () => {
    vi.mocked(useTheme).mockReturnValue({
      theme: "dark",
      setTheme: mockSetTheme,
      resolvedTheme: "dark",
      themes: ["light", "dark"],
    });

    const { container } = render(<ThemeToggle />);

    expect(container.querySelector(".lucide-moon")).toBeTruthy();
  });
});
