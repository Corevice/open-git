import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import NewRepoPage from "@/app/(app)/new/page";

const mockPush = vi.fn();
const mockCreateForUser = vi.fn();
const mockCreateForOrg = vi.fn();
const mockGetCurrent = vi.fn();
const mockGet = vi.fn();
const mockSetToken = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => ({ isAuthenticated: true, token: "test-token" }),
}));

vi.mock("@/lib/api", () => {
  class ApiError extends Error {
    status: number;
    constructor(status: number, message: string) {
      super(message);
      this.name = "ApiError";
      this.status = status;
    }
  }
  return {
    ApiError,
    ApiClient: vi.fn().mockImplementation(() => ({
      setToken: mockSetToken,
      users: { getCurrent: mockGetCurrent },
      get: mockGet,
      repos: {
        createForUser: mockCreateForUser,
        createForOrg: mockCreateForOrg,
      },
    })),
  };
});

describe("NewRepoPage", () => {
  beforeEach(() => {
    vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "http://localhost:8080");
    mockPush.mockClear();
    mockSetToken.mockReset();
    mockCreateForOrg.mockReset();
    mockGetCurrent.mockReset();
    mockGetCurrent.mockResolvedValue({ login: "testuser", avatar_url: "" });
    mockGet.mockReset();
    mockGet.mockResolvedValue([]);
    mockCreateForUser.mockReset();
    mockCreateForUser.mockResolvedValue({ owner: "testuser", name: "my-repo" });
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("passes auto_init: true to createForUser when checkbox is checked", async () => {
    const user = userEvent.setup();

    render(<NewRepoPage />);

    // Wait for owner options to load and the user to be selected.
    await waitFor(() => {
      expect(
        screen.getByRole("option", { name: /testuser/ }),
      ).toBeInTheDocument();
    });

    await user.type(screen.getByLabelText(/リポジトリ名/), "my-repo");
    await user.click(
      screen.getByLabelText(/README でこのリポジトリを初期化する/),
    );
    await user.click(screen.getByRole("button", { name: /リポジトリを作成/ }));

    await waitFor(() => {
      expect(mockCreateForUser).toHaveBeenCalledWith({
        name: "my-repo",
        description: undefined,
        private: false,
        auto_init: true,
      });
    });

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith("/testuser/my-repo");
    });
  });
});
