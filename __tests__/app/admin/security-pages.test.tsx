import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import SecurityAdvisoriesPage from "@/app/admin/security/advisories/page";
import SecurityAuditLogPage from "@/app/admin/security/audit-log/page";

const mockPush = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
  useSearchParams: () => new URLSearchParams("org=acme-corp"),
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => ({
    token: "test-token",
    isAuthenticated: true,
    login: vi.fn(),
    logout: vi.fn(),
  }),
}));

vi.mock("@/components/ui/toast", () => ({
  useToast: () => ({
    success: vi.fn(),
    error: vi.fn(),
  }),
}));

function mockAdminFetch() {
  return vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input);

    if (url.endsWith("/api/v3/user")) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ login: "alice" }),
      };
    }

    if (url.endsWith("/api/v3/orgs/acme-corp/members")) {
      return {
        ok: true,
        status: 200,
        json: async () => [{ login: "alice", role: "admin" }],
      };
    }

    if (url.includes("/api/v3/orgs/acme-corp/security-advisories")) {
      return {
        ok: true,
        status: 200,
        json: async () => [
          {
            id: "1",
            ghsa_id: "GHSA-xxxx-yyyy-zzzz",
            severity: "high",
            summary: "Example advisory",
            state: "open",
            affected_package: "example-pkg",
            repository: {
              owner: { login: "acme-corp" },
              name: "demo",
            },
          },
        ],
      };
    }

    if (url.includes("/api/v3/orgs/acme-corp/audit-log/export")) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ job_id: "export-123" }),
      };
    }

    if (
      url.includes("/api/v3/orgs/acme-corp/audit-log") &&
      !url.includes("/export")
    ) {
      return {
        ok: true,
        status: 200,
        json: async () => [
          {
            id: "log-1",
            actor_login: "alice",
            action: "repo.delete",
            ip_address: "192.168.1.10",
            created_at: "2024-01-01T00:00:00Z",
          },
        ],
      };
    }

    return {
      ok: false,
      status: 404,
      json: async () => ({}),
    };
  });
}

describe("Security admin pages", () => {
  beforeEach(() => {
    mockPush.mockClear();
    vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "http://localhost:8080");
  });

  it("renders advisories for org admins", async () => {
    vi.stubGlobal("fetch", mockAdminFetch());

    render(<SecurityAdvisoriesPage />);

    await waitFor(() => {
      expect(screen.getByText("GHSA-xxxx-yyyy-zzzz")).toBeInTheDocument();
    });
  });

  it("shows access denied for non-admin members on advisories page", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);

        if (url.endsWith("/api/v3/user")) {
          return {
            ok: true,
            status: 200,
            json: async () => ({ login: "bob" }),
          };
        }

        if (url.endsWith("/api/v3/orgs/acme-corp/members")) {
          return {
            ok: true,
            status: 200,
            json: async () => [{ login: "bob", role: "member" }],
          };
        }

        return { ok: false, status: 404, json: async () => ({}) };
      }),
    );

    render(<SecurityAdvisoriesPage />);

    await waitFor(() => {
      expect(screen.getByText("Access Denied")).toBeInTheDocument();
    });
  });

  it("uses action filter param and masks IP addresses on audit log page", async () => {
    const fetchMock = mockAdminFetch();
    vi.stubGlobal("fetch", fetchMock);

    render(<SecurityAuditLogPage />);

    await waitFor(() => {
      expect(screen.getByText("repo.delete")).toBeInTheDocument();
    });

    expect(screen.getByText("192.168.*.*")).toBeInTheDocument();

    await userEvent.selectOptions(
      screen.getByLabelText("Action"),
      "repo.delete",
    );
    await userEvent.click(screen.getByRole("button", { name: "Search" }));

    await waitFor(() => {
      expect(
        fetchMock.mock.calls.some(([url]) => {
          const parsed = new URL(String(url));
          return parsed.searchParams.get("action") === "repo.delete";
        }),
      ).toBe(true);
    });
  });
});
