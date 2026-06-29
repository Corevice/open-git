import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockGetAppMeta = vi.fn();

vi.mock("@/lib/api", () => ({
  getAppMeta: (...args: unknown[]) => mockGetAppMeta(...args),
}));

import AboutPage from "@/app/about/page";

describe("AboutPage", () => {
  beforeEach(() => {
    mockGetAppMeta.mockReset();
    vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "http://localhost:8080");
    mockGetAppMeta.mockResolvedValue({
      app_name: "OpenGit",
      version: "1.0.0",
      git_commit: "abc1234",
      build_date: "2025-01-01T00:00:00Z",
      license: "Apache-2.0",
      source_url: "https://example.org/repo",
    });
  });

  it("renders project name and includes a link", async () => {
    render(<AboutPage />);

    await waitFor(() => {
      expect(screen.getByText("OpenGit")).toBeInTheDocument();
    });
    expect(document.querySelector("a")).not.toBeNull();
  });
});
