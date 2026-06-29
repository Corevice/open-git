import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import OrgPeoplePage from "@/app/(app)/[owner]/people/page";
import { resolvedParams } from "./support/params";

const mockPush = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => ({ token: "test-token", isAuthenticated: true, login: vi.fn(), logout: vi.fn() }),
}));

describe("OrgPeoplePage", () => {
  beforeEach(() => {
    mockPush.mockClear();
    vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "http://localhost:8080");
  });

  it("renders member table rows and invite form for owners", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);

      if (url.endsWith("/api/v3/orgs/acme-corp/members") && init?.method !== "PUT") {
        return {
          ok: true,
          status: 200,
          headers: new Headers({ "content-type": "application/json" }),
          json: async () => [
            { id: 1, login: "alice", role: "owner" },
            { id: 2, login: "bob", role: "member" },
          ],
        };
      }

      if (url.endsWith("/api/v3/user")) {
        return {
          ok: true,
          status: 200,
          headers: new Headers({ "content-type": "application/json" }),
          json: async () => ({
            id: 1,
            login: "alice",
            email: "alice@example.com",
          }),
        };
      }

      return {
        ok: false,
        status: 404,
        statusText: "Not Found",
        json: async () => ({ message: "Not Found" }),
      };
    });

    vi.stubGlobal("fetch", fetchMock);

    render(
      <OrgPeoplePage params={resolvedParams({ owner: "acme-corp" })} />,
    );

    expect(await screen.findByText("alice")).toBeInTheDocument();
    expect(screen.getByText("bob")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Invite member" })).toBeInTheDocument();
    expect(screen.getByLabelText("Username")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Invite member" })).toBeInTheDocument();
  });

  it("shows inline error on 403 invite attempt", async () => {
    const user = userEvent.setup();

    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);

      if (url.endsWith("/api/v3/orgs/acme-corp/members") && init?.method !== "PUT") {
        return {
          ok: true,
          status: 200,
          headers: new Headers({ "content-type": "application/json" }),
          json: async () => [{ id: 1, login: "alice", role: "owner" }],
        };
      }

      if (url.endsWith("/api/v3/user")) {
        return {
          ok: true,
          status: 200,
          headers: new Headers({ "content-type": "application/json" }),
          json: async () => ({
            id: 1,
            login: "alice",
            email: "alice@example.com",
          }),
        };
      }

      if (
        url.endsWith("/api/v3/orgs/acme-corp/memberships/stranger") &&
        init?.method === "PUT"
      ) {
        return {
          ok: false,
          status: 403,
          statusText: "Forbidden",
          json: async () => ({ message: "Forbidden" }),
        };
      }

      return {
        ok: false,
        status: 404,
        statusText: "Not Found",
        json: async () => ({ message: "Not Found" }),
      };
    });

    vi.stubGlobal("fetch", fetchMock);

    render(
      <OrgPeoplePage params={resolvedParams({ owner: "acme-corp" })} />,
    );

    await screen.findByRole("heading", { name: "Invite member" });

    await user.type(screen.getByLabelText("Username"), "stranger");
    await user.click(screen.getByRole("button", { name: "Invite member" }));

    await waitFor(() => {
      expect(screen.getByText("Forbidden")).toBeInTheDocument();
    });
  });
});
