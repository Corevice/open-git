import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import NewRepoPage from "@/app/(app)/new/page";

const mockPush = vi.fn();
const mockCreateRepo = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => ({ isAuthenticated: true, token: "test-token" }),
}));

vi.mock("@/lib/api-client", () => ({
  apiClient: {
    createRepo: (...args: unknown[]) => mockCreateRepo(...args),
  },
}));

describe("NewRepoPage", () => {
  beforeEach(() => {
    mockPush.mockClear();
    mockCreateRepo.mockReset();
    mockCreateRepo.mockResolvedValue({ owner: "testuser", name: "my-repo" });
  });

  it("passes auto_init: true to createRepo when checkbox is checked", async () => {
    const user = userEvent.setup();

    render(<NewRepoPage />);

    await user.type(screen.getByLabelText(/リポジトリ名/), "my-repo");
    await user.click(
      screen.getByRole("checkbox", { name: /リポジトリをREADMEで初期化する/ }),
    );
    await user.click(screen.getByRole("button", { name: /リポジトリを作成/ }));

    await waitFor(() => {
      expect(mockCreateRepo).toHaveBeenCalledWith("my-repo", "public", {
        description: undefined,
        autoInit: true,
      });
    });
  });
});
