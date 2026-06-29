import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import NewIssuePage from "@/app/(app)/[owner]/[repo]/issues/new/page";

const mockPush = vi.fn();
const mockToastSuccess = vi.fn();
const mockToastError = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => ({ isAuthenticated: true, token: "test-token" }),
}));

vi.mock("@/components/ui/toast", () => ({
  useToast: () => ({
    success: mockToastSuccess,
    error: mockToastError,
  }),
}));

describe("NewIssuePage", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    mockPush.mockClear();
    mockToastSuccess.mockClear();
    mockToastError.mockClear();
    vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "http://localhost:8080");

    fetchMock = vi.fn((url: string | URL, init?: RequestInit) => {
      const urlStr = String(url);

      if (init?.method === "POST" && urlStr.includes("/api/v3/repos/acme/demo/issues")) {
        return Promise.resolve({
          ok: true,
          status: 201,
          json: async () => ({ number: 42 }),
        });
      }

      if (urlStr.includes("/labels")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => [],
        });
      }

      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({}),
      });
    });

    vi.stubGlobal("fetch", fetchMock);
  });

  it("shows inline error for empty title without calling issue create fetch", async () => {
    const user = userEvent.setup();

    render(<NewIssuePage params={Promise.resolve({ owner: "acme", repo: "demo" })} />);

    await waitFor(() => {
      expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /submit new issue/i }));

    expect(await screen.findByText("Title is required")).toBeInTheDocument();

    const issuePosts = fetchMock.mock.calls.filter(
      ([url, init]) =>
        String(url).includes("/api/v3/repos/acme/demo/issues") && init?.method === "POST",
    );
    expect(issuePosts).toHaveLength(0);
  });

  it("submits valid title, disables button while submitting, shows toast, and navigates", async () => {
    let resolvePost: (value: unknown) => void = () => {};
    const postPromise = new Promise((resolve) => {
      resolvePost = resolve;
    });

    fetchMock.mockImplementation((url: string | URL, init?: RequestInit) => {
      const urlStr = String(url);

      if (init?.method === "POST" && urlStr.includes("/api/v3/repos/acme/demo/issues")) {
        return postPromise.then(() => ({
          ok: true,
          status: 201,
          json: async () => ({ number: 42 }),
          text: async () => JSON.stringify({ number: 42 }),
        }));
      }

      if (urlStr.includes("/labels")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => [],
        });
      }

      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({}),
      });
    });

    const user = userEvent.setup();

    render(<NewIssuePage params={Promise.resolve({ owner: "acme", repo: "demo" })} />);

    await waitFor(() => {
      expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
    });

    await user.type(screen.getByLabelText(/title/i), "Fix login bug");

    const submitButton = screen.getByRole("button", { name: /submit new issue/i });
    await user.click(submitButton);

    await waitFor(() => {
      expect(submitButton).toBeDisabled();
    });

    resolvePost(undefined);

    await waitFor(() => {
      expect(mockToastSuccess).toHaveBeenCalledWith("Issue created");
      expect(mockPush).toHaveBeenCalledWith("/acme/demo/issues/42");
    });
  });

  it("shows toast error on 422 response", async () => {
    fetchMock.mockImplementation((url: string | URL, init?: RequestInit) => {
      const urlStr = String(url);

      if (init?.method === "POST" && urlStr.includes("/api/v3/repos/acme/demo/issues")) {
        return Promise.resolve({
          ok: false,
          status: 422,
          statusText: "Too long",
          json: async () => ({ errors: [{ field: "title", message: "Too long" }] }),
        });
      }

      if (urlStr.includes("/labels")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => [],
        });
      }

      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({}),
      });
    });

    const user = userEvent.setup();

    render(<NewIssuePage params={Promise.resolve({ owner: "acme", repo: "demo" })} />);

    await waitFor(() => {
      expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
    });

    await user.type(screen.getByLabelText(/title/i), "Valid title");
    await user.click(screen.getByRole("button", { name: /submit new issue/i }));

    await waitFor(() => {
      expect(mockToastError).toHaveBeenCalledWith("Too long");
    });
  });
});
