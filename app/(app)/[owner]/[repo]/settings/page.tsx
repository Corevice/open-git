"use client";

import { FormEvent, use, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  RbacGate,
  type RbacRole,
} from "@/components/layout/RbacGate";
import {
  apiClient as baseApiClient,
  isApiError,
  type ApiError,
} from "@/lib/api-client";
import { ApiClient } from "@/lib/api";
import { AUTH_TOKEN_KEY } from "@/lib/auth";

type RepoSettingsApiClient = typeof baseApiClient & {
  repos: {
    updateRepo(
      owner: string,
      repo: string,
      data: { name: string },
    ): Promise<unknown>;
    deleteRepo(owner: string, repo: string): Promise<void>;
  };
};

const apiBase = (
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  process.env.NEXT_PUBLIC_API_URL ??
  ""
).replace(/\/$/, "");

async function repoRequest<T>(
  method: "PATCH" | "DELETE",
  owner: string,
  repo: string,
  body?: unknown,
): Promise<T> {
  const res = await fetch(`${apiBase}/api/v3/repos/${owner}/${repo}`, {
    method,
    headers: {
      Accept: "application/vnd.github+json",
      "Content-Type": "application/json",
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    let message = res.statusText;
    let code = String(res.status);
    try {
      const errorBody = (await res.json()) as { message?: string; code?: string };
      message = errorBody.message ?? message;
      code = errorBody.code ?? code;
    } catch {
      // ignore non-JSON error bodies
    }
    const error: ApiError = { status: res.status, code, message };
    throw error;
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return (await res.json()) as T;
}

const apiClient = Object.assign(baseApiClient, {
  repos: {
    updateRepo(owner: string, repo: string, data: { name: string }) {
      return repoRequest("PATCH", owner, repo, data);
    },
    deleteRepo(owner: string, repo: string) {
      return repoRequest("DELETE", owner, repo);
    },
  },
}) as RepoSettingsApiClient;

function mapMembershipRoleToRbac(role: string): RbacRole {
  const normalized = role.toLowerCase();
  if (normalized === "owner" || normalized === "admin") {
    return "admin";
  }
  if (normalized === "write" || normalized === "maintainer") {
    return "write";
  }
  return "read";
}

export default function RepoSettingsPage({
  params,
}: {
  params: Promise<{ owner: string; repo: string }>;
}) {
  const { owner, repo } = use(params);
  const router = useRouter();

  const membershipClient = useMemo(() => {
    const baseURL =
      process.env.NEXT_PUBLIC_API_BASE_URL ??
      process.env.NEXT_PUBLIC_API_URL ??
      "";
    const token =
      typeof window !== "undefined"
        ? localStorage.getItem(AUTH_TOKEN_KEY)
        : null;
    const client = new ApiClient(baseURL);
    if (token) {
      client.setToken(token);
    }
    return client;
  }, []);

  const [userRole, setUserRole] = useState<RbacRole | null>(null);
  const [newName, setNewName] = useState(repo);
  const [renameSubmitting, setRenameSubmitting] = useState(false);
  const [renameError, setRenameError] = useState<string | null>(null);

  const [showDeleteForm, setShowDeleteForm] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState("");
  const [deleteSubmitting, setDeleteSubmitting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const fullName = `${owner}/${repo}`;
  const nameUnchanged = newName.trim() === repo;
  const deleteConfirmed = deleteConfirm === fullName;

  useEffect(() => {
    setNewName(repo);
  }, [repo]);

  useEffect(() => {
    let cancelled = false;

    async function loadUserRole() {
      try {
        const currentUser = await membershipClient.users.getCurrent();
        if (cancelled) {
          return;
        }

        if (currentUser.login === owner) {
          setUserRole("admin");
          return;
        }

        const members = await membershipClient.orgs.listMembers(owner);
        if (cancelled) {
          return;
        }

        const membership = members.find(
          (member) => member.login === currentUser.login,
        );
        setUserRole(
          membership ? mapMembershipRoleToRbac(membership.role) : null,
        );
      } catch {
        if (!cancelled) {
          setUserRole(null);
        }
      }
    }

    void loadUserRole();

    return () => {
      cancelled = true;
    };
  }, [membershipClient, owner]);

  const handleRename = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (nameUnchanged || renameSubmitting) return;

    setRenameError(null);
    setRenameSubmitting(true);
    try {
      const trimmedName = newName.trim();
      await apiClient.repos.updateRepo(owner, repo, { name: trimmedName });
      router.push(`/${owner}/${trimmedName}/settings`);
    } catch (error) {
      if (isApiError(error) && error.status === 422) {
        setRenameError(error.message);
      } else if (isApiError(error)) {
        setRenameError(error.message);
      } else {
        setRenameError("Failed to rename repository.");
      }
    } finally {
      setRenameSubmitting(false);
    }
  };

  const handleDelete = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!deleteConfirmed || deleteSubmitting) return;

    setDeleteError(null);
    setDeleteSubmitting(true);
    try {
      await apiClient.repos.deleteRepo(owner, repo);
      router.push("/dashboard");
    } catch (error) {
      if (isApiError(error)) {
        setDeleteError(error.message);
      } else {
        setDeleteError("Failed to delete repository.");
      }
    } finally {
      setDeleteSubmitting(false);
    }
  };

  return (
    <div className="mx-auto max-w-3xl space-y-8 px-6 py-8">
      <h1 className="text-2xl font-semibold">Settings</h1>
      <p className="text-sm text-[#656d76]">
        {owner}/{repo}
      </p>

      <RbacGate requiredRole="admin" userRole={userRole}>
        <section className="rounded-md border border-[#d0d7de] bg-white p-5">
          <h2 className="text-lg font-semibold">Rename repository</h2>
          <form onSubmit={handleRename} className="mt-4 space-y-4">
            <div>
              <label
                htmlFor="rename-repo"
                className="mb-1.5 block text-sm font-semibold"
              >
                Repository name
              </label>
              <input
                id="rename-repo"
                type="text"
                value={newName}
                onChange={(event) => setNewName(event.target.value)}
                className="w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm"
              />
            </div>
            {renameError && (
              <p className="text-sm text-[#cf222e]">{renameError}</p>
            )}
            <button
              type="submit"
              disabled={nameUnchanged || renameSubmitting}
              className="rounded-md bg-[#1f883d] px-4 py-2 text-sm font-semibold text-white hover:bg-[#1a7f37] disabled:opacity-50"
            >
              {renameSubmitting ? "Renaming…" : "Rename repository"}
            </button>
          </form>
        </section>
      </RbacGate>

      <RbacGate requiredRole="admin" userRole={userRole}>
        <div className="rounded-md border border-[#cf222e] bg-white p-5">
          <h2 className="text-lg font-semibold text-[#cf222e]">Danger Zone</h2>
          <p className="mt-2 text-sm text-[#656d76]">
            Once deleted, there is no going back. Please be certain.
          </p>

          {!showDeleteForm ? (
            <button
              type="button"
              onClick={() => setShowDeleteForm(true)}
              className="mt-4 rounded-md border border-[#cf222e] bg-white px-4 py-2 text-sm font-semibold text-[#cf222e] hover:bg-[#ffebe9]"
            >
              Delete this repository
            </button>
          ) : (
            <form onSubmit={handleDelete} className="mt-4 space-y-4">
              <div>
                <label
                  htmlFor="delete-confirm"
                  className="mb-1.5 block text-sm font-semibold"
                >
                  Type owner/repo to confirm
                </label>
                <input
                  id="delete-confirm"
                  type="text"
                  value={deleteConfirm}
                  onChange={(event) => setDeleteConfirm(event.target.value)}
                  className="w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm font-mono"
                />
              </div>
              {deleteError && (
                <p className="text-sm text-[#cf222e]">{deleteError}</p>
              )}
              <button
                type="submit"
                disabled={!deleteConfirmed || deleteSubmitting}
                className="rounded-md border border-[#cf222e] bg-[#cf222e] px-4 py-2 text-sm font-semibold text-white hover:bg-[#a40e26] disabled:opacity-50"
              >
                {deleteSubmitting ? "Deleting…" : "Delete this repository"}
              </button>
            </form>
          )}
        </div>
      </RbacGate>
    </div>
  );
}
