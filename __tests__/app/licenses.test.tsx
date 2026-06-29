import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockGetAppLicenses = vi.fn();

vi.mock("@/lib/api", () => ({
  getAppLicenses: (...args: unknown[]) => mockGetAppLicenses(...args),
}));

import LicensesPage from "@/app/licenses/page";

describe("LicensesPage", () => {
  beforeEach(() => {
    mockGetAppLicenses.mockReset();
    vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "http://localhost:8080");
  });

  it("renders third-party license entries", async () => {
    mockGetAppLicenses.mockResolvedValue({
      app_license: "Apache-2.0",
      third_party: [
        {
          name: "react",
          version: "18.0.0",
          license: "MIT",
          url: "https://react.dev",
        },
        {
          name: "next",
          version: "14.0.0",
          license: "MIT",
          url: "https://nextjs.org",
        },
      ],
    });

    render(<LicensesPage />);

    await waitFor(() => {
      expect(screen.getByText("react")).toBeInTheDocument();
      expect(screen.getByText("next")).toBeInTheDocument();
    });
  });

  it("shows fallback text when third_party is empty", async () => {
    mockGetAppLicenses.mockResolvedValue({
      app_license: "Apache-2.0",
      third_party: [],
    });

    render(<LicensesPage />);

    await waitFor(() => {
      expect(
        screen.getByText("ライセンス情報を取得できませんでした"),
      ).toBeInTheDocument();
    });
  });
});
