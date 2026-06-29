import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const { mockBranding, mockEnv } = vi.hoisted(() => ({
  mockBranding: {
    appName: "TestApp",
    logoSrc: "/brand/logo.svg",
    faviconSrc: "/brand/favicon.ico",
    primaryColor: "#1f6feb",
    sourceUrl: "https://example.com/org/repo",
    licenseName: "MIT",
  },
  mockEnv: {
    NEXT_PUBLIC_APP_VERSION: "1.2.3",
  },
}));

vi.mock("@/lib/branding", () => ({
  BRANDING: mockBranding,
}));

vi.mock("@/lib/env", () => ({
  env: mockEnv,
}));

import { Footer } from "@/components/layout/Footer";

describe("Footer", () => {
  it("renders license name, source link, and version", () => {
    mockEnv.NEXT_PUBLIC_APP_VERSION = "1.2.3";

    render(<Footer />);

    expect(screen.getByText(mockBranding.licenseName)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Source code" })).toHaveAttribute(
      "href",
      mockBranding.sourceUrl,
    );
    expect(screen.getByText("1.2.3")).toBeInTheDocument();
  });

  it('renders "dev" when version is dev', () => {
    mockEnv.NEXT_PUBLIC_APP_VERSION = "dev";

    render(<Footer />);

    expect(screen.getByText("dev")).toBeInTheDocument();
  });
});
