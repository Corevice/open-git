"use client";

import { FormEvent, use, useEffect, useMemo, useState } from "react";

import { ApiClient, ApiError } from "@/lib/api";
import type { OrgMember, User } from "@/lib/api-types";
import { useAuth } from "@/lib/auth";

type OrgRole = "owner" | "member";

type OrgPeopleApiClient = Omit<ApiClient, "orgs"> & {
  orgs: Omit<ApiClient["orgs"], "inviteMember" | "removeMember"> & {
    inviteMember(
      org: string,
      username: string,
      role: OrgRole,
    ): Promise<OrgMember>;
    removeMember(org: string, username: string): Promise<void>;
  };
};

function createOrgPeopleClient(baseURL: string, token: string | null): OrgPeopleApiClient {
  const client = new ApiClient(baseURL);
  if (token) {
    client.setToken(token);
  }

  const apiBase = baseURL.replace(/\/$/, "");

  async function orgRequest<T>(
    method: "PUT" | "DELETE",
    path: string,
    body?: unknown,
  ): Promise<T> {
    const headers: Record<string, string> = {
      Accept: "application/vnd.github+json",
      "Content-Type": "application/json",
    };
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }

    const res = await fetch(`${apiBase}${path}`, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });

    if (!res.ok) {
      let message = res.statusText;
      try {
        const errorBody = (await res.json()) as { message?: string };
        message = errorBody.message ?? message;
      } catch {
        // ignore non-JSON error bodies
      }
      throw new ApiError(res.status, message);
    }

    if (res.status === 204) {
      return undefined as T;
    }

    return (await res.json()) as T;
  }

  return Object.assign(client, {
    orgs: {
      ...client.orgs,
      inviteMember(org: string, username: string, role: OrgRole) {
        return orgRequest<OrgMember>(
          "PUT",
          `/api/v3/orgs/${org}/memberships/${username}`,
          { role },
        );
      },
      removeMember(org: string, username: string) {
        return orgRequest<void>(
          "DELETE",
          `/api/v3/orgs/${org}/members/${username}`,
        );
      },
    },
  });
}

export default function OrgPeoplePage({
  params,
}: {
  params: Promise<{ owner: string }>;
}) {
  const { owner } = use(params);
  const { token } = useAuth();

  const baseURL =
    process.env.NEXT_PUBLIC_API_BASE_URL ??
    process.env.NEXT_PUBLIC_API_URL ??
    "http://localhost:8080";

  const apiClient = useMemo(
    () => createOrgPeopleClient(baseURL, token),
    [baseURL, token],
  );

  const [members, setMembers] = useState<OrgMember[]>([]);
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [inviteUsername, setInviteUsername] = useState("");
  const [inviteRole, setInviteRole] = useState<OrgRole>("member");
  const [inviteSubmitting, setInviteSubmitting] = useState(false);
  const [inviteError, setInviteError] = useState<string | null>(null);

  const [rowError, setRowError] = useState<string | null>(null);
  const [updatingLogin, setUpdatingLogin] = useState<string | null>(null);
  const [removingLogin, setRemovingLogin] = useState<string | null>(null);

  const isOwner =
    currentUser !== null &&
    members.some(
      (member) => member.login === currentUser.login && member.role === "owner",
    );

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setLoadError(null);

      try {
        const [memberList, viewer] = await Promise.all([
          apiClient.orgs.listMembers(owner),
          apiClient.users.getCurrent(),
        ]);

        if (!cancelled) {
          setMembers(memberList);
          setCurrentUser(viewer);
        }
      } catch (error) {
        if (!cancelled) {
          setLoadError(
            error instanceof Error ? error.message : "Failed to load members.",
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [apiClient, owner]);

  const handleInvite = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const username = inviteUsername.trim();
    if (!username || inviteSubmitting) return;

    setInviteError(null);
    setInviteSubmitting(true);
    try {
      const member = await apiClient.orgs.inviteMember(owner, username, inviteRole);
      setMembers((prev) => {
        const existing = prev.find((m) => m.login === member.login);
        if (existing) {
          return prev.map((m) =>
            m.login === member.login ? { ...m, role: member.role } : m,
          );
        }
        return [...prev, member];
      });
      setInviteUsername("");
      setInviteRole("member");
    } catch (error) {
      if (error instanceof ApiError && (error.status === 403 || error.status === 422)) {
        setInviteError(error.message);
      } else if (error instanceof ApiError) {
        setInviteError(error.message);
      } else {
        setInviteError("Failed to invite member.");
      }
    } finally {
      setInviteSubmitting(false);
    }
  };

  const handleRoleChange = async (username: string, role: OrgRole) => {
    setRowError(null);
    setUpdatingLogin(username);
    try {
      const updated = await apiClient.orgs.inviteMember(owner, username, role);
      setMembers((prev) =>
        prev.map((member) =>
          member.login === username ? { ...member, role: updated.role } : member,
        ),
      );
    } catch (error) {
      if (error instanceof ApiError && (error.status === 403 || error.status === 422)) {
        setRowError(error.message);
      } else if (error instanceof ApiError) {
        setRowError(error.message);
      } else {
        setRowError("Failed to update member role.");
      }
    } finally {
      setUpdatingLogin(null);
    }
  };

  const handleRemove = async (username: string) => {
    if (!window.confirm(`Remove ${username} from this organization?`)) {
      return;
    }

    setRowError(null);
    setRemovingLogin(username);
    try {
      await apiClient.orgs.removeMember(owner, username);
      setMembers((prev) => prev.filter((member) => member.login !== username));
    } catch (error) {
      if (error instanceof ApiError && (error.status === 403 || error.status === 422)) {
        setRowError(error.message);
      } else if (error instanceof ApiError) {
        setRowError(error.message);
      } else {
        setRowError("Failed to remove member.");
      }
    } finally {
      setRemovingLogin(null);
    }
  };

  if (loading) {
    return (
      <div className="mx-auto max-w-4xl px-6 py-8">
        <p className="text-sm text-[#656d76]">Loading members…</p>
      </div>
    );
  }

  if (loadError) {
    return (
      <div className="mx-auto max-w-4xl px-6 py-8">
        <p className="text-sm text-[#cf222e]">{loadError}</p>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl space-y-8 px-6 py-8">
      <div>
        <h1 className="text-2xl font-semibold">People</h1>
        <p className="mt-1 text-sm text-[#656d76]">{owner}</p>
      </div>

      {isOwner ? (
        <section className="rounded-md border border-[#d0d7de] bg-white p-5">
          <h2 className="text-lg font-semibold">Invite member</h2>
          <form onSubmit={handleInvite} className="mt-4 flex flex-wrap items-end gap-4">
            <div>
              <label
                htmlFor="invite-username"
                className="mb-1.5 block text-sm font-semibold"
              >
                Username
              </label>
              <input
                id="invite-username"
                type="text"
                value={inviteUsername}
                onChange={(event) => setInviteUsername(event.target.value)}
                className="rounded-md border border-[#d0d7de] px-3 py-2 text-sm"
              />
            </div>
            <div>
              <label
                htmlFor="invite-role"
                className="mb-1.5 block text-sm font-semibold"
              >
                Role
              </label>
              <select
                id="invite-role"
                value={inviteRole}
                onChange={(event) => setInviteRole(event.target.value as OrgRole)}
                className="rounded-md border border-[#d0d7de] px-3 py-2 text-sm"
              >
                <option value="owner">owner</option>
                <option value="member">member</option>
              </select>
            </div>
            <button
              type="submit"
              disabled={!inviteUsername.trim() || inviteSubmitting}
              className="rounded-md bg-[#1f883d] px-4 py-2 text-sm font-semibold text-white hover:bg-[#1a7f37] disabled:opacity-50"
            >
              {inviteSubmitting ? "Inviting…" : "Invite member"}
            </button>
          </form>
          {inviteError ? (
            <p className="mt-3 text-sm text-[#cf222e]">{inviteError}</p>
          ) : null}
        </section>
      ) : null}

      <section className="rounded-md border border-[#d0d7de] bg-white p-5">
        <h2 className="text-lg font-semibold">Members</h2>
        {rowError ? (
          <p className="mt-3 text-sm text-[#cf222e]">{rowError}</p>
        ) : null}
        <div className="mt-4 overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-[#d0d7de]">
                <th className="py-2 pr-4 font-semibold">Login</th>
                <th className="py-2 pr-4 font-semibold">Role</th>
                {isOwner ? <th className="py-2 font-semibold">Actions</th> : null}
              </tr>
            </thead>
            <tbody>
              {members.map((member) => (
                <tr key={member.login} className="border-b border-[#d0d7de]">
                  <td className="py-3 pr-4 font-mono">{member.login}</td>
                  <td className="py-3 pr-4">
                    {isOwner ? (
                      <select
                        value={member.role}
                        disabled={updatingLogin === member.login}
                        onChange={(event) =>
                          void handleRoleChange(
                            member.login,
                            event.target.value as OrgRole,
                          )
                        }
                        className="rounded-md border border-[#d0d7de] px-2 py-1 text-sm"
                        aria-label={`Role for ${member.login}`}
                      >
                        <option value="owner">owner</option>
                        <option value="member">member</option>
                      </select>
                    ) : (
                      member.role
                    )}
                  </td>
                  {isOwner ? (
                    <td className="py-3">
                      <button
                        type="button"
                        onClick={() => void handleRemove(member.login)}
                        disabled={removingLogin === member.login}
                        className="rounded-md border border-[#cf222e] px-3 py-1 text-sm font-semibold text-[#cf222e] hover:bg-[#ffebe9] disabled:opacity-50"
                      >
                        {removingLogin === member.login ? "Removing…" : "Remove"}
                      </button>
                    </td>
                  ) : null}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
