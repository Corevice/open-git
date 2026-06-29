import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockGetAppMeta = vi.fn();

vi.mock("@/lib/api", () => ({
  getAppMeta: (...args: unknown[]) => mockGetAppMeta(...args),
}));

import AboutPage from "@/app/about/page";
import { BRANDING } from "@/lib/branding";

const fixedMeta = {
  app_name: "OpenGit",
  version: "1.2.3",
  git_commit: "abc1234",
  build_date: "2025-01-01T00:00:00Z",
  license: "Apache-2.0",
  source_url: "https://example.org/repo",
};

describe("AboutPage", () => {
  beforeEach(() => {
    mockGetAppMeta.mockReset();
    vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "http://localhost:8080");
  });

  it("shows loading state before promise resolves", () => {
    mockGetAppMeta.mockReturnValue(new Promise(() => {}));

    render(<AboutPage />);

    expect(screen.getByText("Loading...")).toBeInTheDocument();
    expect(screen.getByText("dev")).toBeInTheDocument();
  });

  it("renders app name, version, and link to licenses", async () => {
    mockGetAppMeta.mockResolvedValue(fixedMeta);

    render(<AboutPage />);

    await waitFor(() => {
      expect(screen.getByRole("heading", { name: BRANDING.appName })).toBeInTheDocument();
      expect(screen.getByText("1.2.3")).toBeInTheDocument();
      expect(screen.getByRole("link", { name: "Apache-2.0" })).toHaveAttribute(
        "href",
        "/licenses",
      );
    });
  });
});
