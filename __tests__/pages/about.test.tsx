import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import AboutPage from "@/app/about/page";

describe("AboutPage", () => {
  beforeEach(() => {
    vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "http://localhost:8080");

    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url.endsWith("/api/v1/version")) {
          return {
            ok: true,
            status: 200,
            json: async () => ({
              data: {
                version: "1.0.0",
                commit: "abc1234",
                buildDate: "2025-01-01T00:00:00Z",
              },
            }),
          };
        }
        return {
          ok: false,
          status: 404,
          json: async () => ({}),
        };
      }),
    );
  });

  it("renders project name and includes a link", async () => {
    const page = await AboutPage();
    render(page);

    expect(screen.getByText("OpenGit")).toBeInTheDocument();
    expect(document.querySelector("a")).not.toBeNull();
  });
});
