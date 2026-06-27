"use client";

import { Monitor, Moon, Sun } from "lucide-react";
import { useTheme } from "next-themes";

export function ThemeToggle() {
  const { theme, setTheme, resolvedTheme } = useTheme();

  const cycleTheme = () => {
    if (theme === "light") {
      setTheme("dark");
    } else if (theme === "dark") {
      setTheme("system");
    } else {
      setTheme("light");
    }
  };

  const Icon =
    theme === "system"
      ? Monitor
      : resolvedTheme === "dark"
        ? Moon
        : Sun;

  return (
    <button
      type="button"
      onClick={cycleTheme}
      aria-label="Toggle theme"
      className="inline-flex size-9 items-center justify-center rounded-md text-[#c9d1d9] transition-colors hover:bg-[#30363d] hover:text-white"
    >
      <Icon className="size-5" />
    </button>
  );
}
