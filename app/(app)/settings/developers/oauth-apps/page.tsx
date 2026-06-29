"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { ApiClient } from "@/lib/api";
import type { OAuthApp } from "@/lib/api-types";
import { useAuth } from "@/lib/auth";
import { env } from "@/lib/env";

function formatDate(value: string | null): string {
  if (!value) {
    return "—";
  }
  return new Date(value).toLocaleString();
}

export default function OAuthAppsListPage() {
  const { token } = useAuth();
  const router = useRouter();
  const apiClient = useMemo(
    () => new ApiClient(env.NEXT_PUBLIC_API_BASE_URL, router),
    [router],
  );

  const [apps, setApps] = useState<OAuthApp[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  useEffect(() => {
    if (token) {
      apiClient.setToken(token);
    }
  }, [apiClient, token]);

  useEffect(() => {
    if (!token) {
      setLoading(false);
      return;
    }

    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(null);
      try {
        const list = await apiClient.oauthApps.list();
        if (!cancelled) {
          setApps(list);
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error ? err.message : "Failed to load OAuth apps.",
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
  }, [apiClient, token]);

  const handleDelete = async (app: OAuthApp) => {
    if (
      !window.confirm(
        `Delete OAuth app "${app.name}"? This will permanently revoke all associated access tokens and cannot be undone.`,
      )
    ) {
      return;
    }

    setDeletingId(app.id);
    setError(null);

    try {
      await apiClient.oauthApps.delete(app.id);
      setApps((prev) => prev.filter((item) => item.id !== app.id));
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to delete OAuth app.",
      );
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <main className="mx-auto max-w-[960px] px-6 py-8">
      <div className="mb-6 flex items-start justify-between border-b border-[#d1d9e0] pb-4">
        <div>
          <h1 className="mb-2 text-2xl font-semibold">OAuth Apps</h1>
          <p className="text-sm text-[#59636e]">
            Manage OAuth applications you own. Client secrets are shown only at
            creation or regeneration.
          </p>
        </div>
        <Button asChild>
          <Link href="/settings/developers/oauth-apps/new">New OAuth App</Link>
        </Button>
      </div>

      {error && (
        <p className="mb-4 text-sm text-[#cf222e]" role="alert">
          {error}
        </p>
      )}

      {loading ? (
        <p className="text-sm text-[#59636e]">Loading…</p>
      ) : apps.length === 0 ? (
        <p className="text-sm text-[#59636e]">No OAuth apps yet.</p>
      ) : (
        <div className="overflow-x-auto rounded-lg border border-[#d1d9e0]">
          <table className="min-w-full text-left text-sm">
            <thead className="border-b border-[#d1d9e0] bg-[#f6f8fa]">
              <tr>
                <th className="px-4 py-3 font-semibold">Name</th>
                <th className="px-4 py-3 font-semibold">Client ID</th>
                <th className="px-4 py-3 font-semibold">Created</th>
                <th className="px-4 py-3 font-semibold">Actions</th>
              </tr>
            </thead>
            <tbody>
              {apps.map((app) => (
                <tr key={app.id} className="border-b border-[#d1d9e0] last:border-b-0">
                  <td className="px-4 py-3 font-medium">{app.name}</td>
                  <td className="px-4 py-3 font-mono text-xs">{app.client_id}</td>
                  <td className="px-4 py-3">{formatDate(app.created_at)}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-3">
                      <Link
                        href={`/settings/developers/oauth-apps/${app.id}`}
                        className="text-[#0969da] hover:underline"
                      >
                        Edit
                      </Link>
                      <Button
                        type="button"
                        variant="destructive"
                        size="sm"
                        onClick={() => handleDelete(app)}
                        disabled={deletingId === app.id}
                      >
                        {deletingId === app.id ? "Deleting…" : "Delete"}
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </main>
  );
}
