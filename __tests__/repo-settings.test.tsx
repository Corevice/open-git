import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import RepoSettingsPage from "@/app/(app)/[owner]/[repo]/settings/page";
import { resolvedParams } from "./support/params";

const mockPush = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => ({
    token: "test-token",
    isAuthenticated: true,
    login: vi.fn(),
    logout: vi.fn(),
  }),
}));

describe("RepoSettingsPage", () => {
  beforeEach(() => {
    mockPush.mockClear();
    vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "http://localhost:8080");
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        const headers = new Headers({ "content-type": "application/json" });

        if (url.endsWith("/api/v3/user")) {
          return {
            ok: true,
            status: 200,
            headers,
            json: async () => ({ id: 1, login: "alice" }),
          };
        }

        if (url.includes("/api/v3/repos/alice/hello")) {
          return {
            ok: true,
            status: 200,
            headers,
            json: async () => ({
              name: "hello",
              owner: { login: "alice" },
            }),
          };
        }

        return {
          ok: true,
          status: 200,
          headers,
          json: async () => ({}),
        };
      }),
    );
  });

  it("shows rename input pre-filled and delete confirmation behavior", async () => {
    const user = userEvent.setup();

    render(
      <RepoSettingsPage
        params={resolvedParams({ owner: "alice", repo: "hello" })}
      />,
    );

    const renameInput = await screen.findByLabelText("Repository name");
    expect(renameInput).toBeInTheDocument();
    expect(renameInput).toHaveValue("hello");

    await user.click(
      screen.getByRole("button", { name: "Delete this repository" }),
    );

    const deleteSubmit = screen.getByRole("button", {
      name: "Delete this repository",
    });
    expect(deleteSubmit).toBeDisabled();

    await user.type(
      screen.getByLabelText("Type owner/repo to confirm"),
      "alice/hello",
    );

    expect(deleteSubmit).toBeEnabled();
  });
});
