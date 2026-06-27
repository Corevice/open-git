import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import RepoSettingsPage from "@/app/(app)/[owner]/[repo]/settings/page";

const mockPush = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
}));

describe("RepoSettingsPage", () => {
  beforeEach(() => {
    mockPush.mockClear();
    vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "http://localhost:8080");
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({}),
      }),
    );
  });

  it("shows rename input pre-filled and delete confirmation behavior", async () => {
    const user = userEvent.setup();

    render(
      <RepoSettingsPage
        params={Promise.resolve({ owner: "alice", repo: "hello" })}
      />,
    );

    const renameInput = screen.getByLabelText("Repository name");
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
