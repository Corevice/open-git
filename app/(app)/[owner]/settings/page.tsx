"use client";

import { FormEvent, use, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import { ApiClient, ApiError } from "@/lib/api";
import type { OrgProfile } from "@/lib/api-types";
import { useAuth } from "@/lib/auth";

type OrgSettingsApiClient = ApiClient & {
  orgs: ApiClient["orgs"] & {
    update(
      org: string,
      data: { name: string; description: string },
    ): Promise<OrgProfile>;
    delete(org: string): Promise<void>;
  };
};

function createOrgSettingsClient(
  baseURL: string,
  token: string | null,
  router: ReturnType<typeof useRouter>,
): OrgSettingsApiClient {
  const client = new ApiClient(baseURL, router);
  if (token) {
    client.setToken(token);
  }

  const apiBase = baseURL.replace(/\/$/, "");

  async function orgRequest<T>(
    method: "PATCH" | "DELETE",
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
      update(org: string, data: { name: string; description: string }) {
        return orgRequest<OrgProfile>("PATCH", `/api/v3/orgs/${org}`, data);
      },
      delete(org: string) {
        return orgRequest<void>("DELETE", `/api/v3/orgs/${org}`);
      },
    },
  });
}

export default function OrgSettingsPage({
  params,
}: {
  params: Promise<{ owner: string }>;
}) {
  const { owner } = use(params);
  const router = useRouter();
  const { token } = useAuth();

  const baseURL =
    process.env.NEXT_PUBLIC_API_BASE_URL ??
    process.env.NEXT_PUBLIC_API_URL ??
    "http://localhost:8080";

  const apiClient = useMemo(
    () => createOrgSettingsClient(baseURL, token, router),
    [baseURL, token, router],
  );

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [submitSuccess, setSubmitSuccess] = useState(false);

  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState("");
  const [deleteSubmitting, setDeleteSubmitting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const deleteConfirmed = deleteConfirm === owner;

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setLoadError(null);

      try {
        const org = await apiClient.orgs.get(owner);
        if (!cancelled) {
          setName(org.name ?? org.login);
          setDescription(org.description ?? "");
        }
      } catch (error) {
        if (!cancelled) {
          setLoadError(
            error instanceof Error ? error.message : "Failed to load organization.",
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

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting) return;

    setSubmitError(null);
    setSubmitSuccess(false);
    setSubmitting(true);
    try {
      await apiClient.orgs.update(owner, {
        name: name.trim(),
        description: description.trim(),
      });
      setSubmitSuccess(true);
    } catch (error) {
      if (error instanceof ApiError) {
        setSubmitError(error.message);
      } else {
        setSubmitError("Failed to update organization.");
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!deleteConfirmed || deleteSubmitting) return;

    setDeleteError(null);
    setDeleteSubmitting(true);
    try {
      await apiClient.orgs.delete(owner);
      router.push("/dashboard");
    } catch (error) {
      if (error instanceof ApiError) {
        setDeleteError(error.message);
      } else {
        setDeleteError("Failed to delete organization.");
      }
    } finally {
      setDeleteSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="mx-auto max-w-3xl px-6 py-8">
        <p className="text-sm text-[#656d76]">Loading settings…</p>
      </div>
    );
  }

  if (loadError) {
    return (
      <div className="mx-auto max-w-3xl px-6 py-8">
        <p className="text-sm text-[#cf222e]">{loadError}</p>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-3xl space-y-8 px-6 py-8">
      <div>
        <h1 className="text-2xl font-semibold">Organization settings</h1>
        <p className="mt-1 text-sm text-[#656d76]">{owner}</p>
      </div>

      <section className="rounded-md border border-[#d0d7de] bg-white p-5">
        <h2 className="text-lg font-semibold">Profile</h2>
        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
          <div>
            <label htmlFor="org-name" className="mb-1.5 block text-sm font-semibold">
              Name
            </label>
            <input
              id="org-name"
              type="text"
              value={name}
              onChange={(event) => setName(event.target.value)}
              className="w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label
              htmlFor="org-description"
              className="mb-1.5 block text-sm font-semibold"
            >
              Description
            </label>
            <textarea
              id="org-description"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              rows={4}
              className="w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm"
            />
          </div>
          {submitError ? (
            <p className="text-sm text-[#cf222e]">{submitError}</p>
          ) : null}
          {submitSuccess ? (
            <p className="text-sm text-[#1a7f37]">Settings saved.</p>
          ) : null}
          <button
            type="submit"
            disabled={submitting}
            className="rounded-md bg-[#1f883d] px-4 py-2 text-sm font-semibold text-white hover:bg-[#1a7f37] disabled:opacity-50"
          >
            {submitting ? "Saving…" : "Save changes"}
          </button>
        </form>
      </section>

      <div className="rounded-md border border-[#cf222e] bg-white p-5">
        <h2 className="text-lg font-semibold text-[#cf222e]">Danger Zone</h2>
        <p className="mt-2 text-sm text-[#656d76]">
          Once deleted, there is no going back. Please be certain.
        </p>

        {!showDeleteDialog ? (
          <button
            type="button"
            onClick={() => setShowDeleteDialog(true)}
            className="mt-4 rounded-md border border-[#cf222e] bg-white px-4 py-2 text-sm font-semibold text-[#cf222e] hover:bg-[#ffebe9]"
          >
            Delete this organization
          </button>
        ) : (
          <div
            role="dialog"
            aria-labelledby="delete-org-title"
            className="mt-4 space-y-4 rounded-md border border-[#d0d7de] bg-[#f6f8fa] p-4"
          >
            <h3 id="delete-org-title" className="text-base font-semibold">
              Confirm organization deletion
            </h3>
            <form onSubmit={handleDelete} className="space-y-4">
              <div>
                <label
                  htmlFor="delete-org-confirm"
                  className="mb-1.5 block text-sm font-semibold"
                >
                  Type the organization login to confirm
                </label>
                <input
                  id="delete-org-confirm"
                  type="text"
                  value={deleteConfirm}
                  onChange={(event) => setDeleteConfirm(event.target.value)}
                  className="w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm font-mono"
                />
              </div>
              {deleteError ? (
                <p className="text-sm text-[#cf222e]">{deleteError}</p>
              ) : null}
              <div className="flex gap-3">
                <button
                  type="submit"
                  disabled={!deleteConfirmed || deleteSubmitting}
                  className="rounded-md border border-[#cf222e] bg-[#cf222e] px-4 py-2 text-sm font-semibold text-white hover:bg-[#a40e26] disabled:opacity-50"
                >
                  {deleteSubmitting ? "Deleting…" : "Delete this organization"}
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setShowDeleteDialog(false);
                    setDeleteConfirm("");
                    setDeleteError(null);
                  }}
                  className="rounded-md border border-[#d0d7de] bg-white px-4 py-2 text-sm font-semibold hover:bg-[#f6f8fa]"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        )}
      </div>
    </div>
  );
}
