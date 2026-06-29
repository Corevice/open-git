"use client";

import { use, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  deleteWebhook,
  formatEvents,
  lastDeliveryLabel,
  listWebhooks,
  type Webhook,
} from "@/lib/api/webhooks";

type WebhookWithDeliveryCount = Webhook & {
  deliveries_count_24h?: number;
};

function ActiveSwitch({ active }: { active: boolean }) {
  return (
    <span
      className={`relative inline-block h-5 w-10 rounded-full ${
        active ? "bg-[#1f883d]" : "bg-[#d0d7de]"
      }`}
      aria-label={active ? "Active" : "Inactive"}
    >
      <span
        className={`absolute top-0.5 left-0.5 h-4 w-4 rounded-full bg-white transition-transform ${
          active ? "translate-x-5" : ""
        }`}
      />
    </span>
  );
}

export default function WebhooksListPage({
  params,
}: {
  params: Promise<{ owner: string; repo: string }>;
}) {
  const { owner, repo } = use(params);
  const [webhooks, setWebhooks] = useState<WebhookWithDeliveryCount[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Webhook | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const loadWebhooks = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await listWebhooks(owner, repo);
      setWebhooks(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load webhooks.");
    } finally {
      setLoading(false);
    }
  }, [owner, repo]);

  useEffect(() => {
    loadWebhooks();
  }, [loadWebhooks]);

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    setDeleteError(null);
    try {
      await deleteWebhook(owner, repo, deleteTarget.id);
      setDeleteTarget(null);
      await loadWebhooks();
    } catch (err) {
      setDeleteError(
        err instanceof Error ? err.message : "Failed to delete webhook.",
      );
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <header className="sticky top-0 z-50 flex h-16 items-center justify-between border-b border-[#d1d9e0] bg-white/85 px-6 backdrop-blur">
        <Link
          href="/dashboard"
          className="flex items-center gap-2 text-lg font-extrabold"
        >
          <span className="text-xl">🐙</span>
          <span>OpenHub</span>
        </Link>
        <Link
          href={`/${owner}/${repo}/settings`}
          className="rounded-md px-3 py-1.5 text-sm hover:bg-[#f6f8fa]"
        >
          ← Settings
        </Link>
      </header>

      <div className="mx-auto max-w-[1200px] px-6 py-6">
        <div className="mb-4 text-sm text-[#656d76]">
          <Link href={`/${owner}/${repo}`} className="text-[#0969da]">
            {owner}/{repo}
          </Link>{" "}
          /{" "}
          <Link
            href={`/${owner}/${repo}/settings`}
            className="text-[#0969da]"
          >
            Settings
          </Link>{" "}
          / Webhooks
        </div>

        <div className="mb-4 flex items-center justify-between">
          <h1 className="text-2xl font-semibold">Webhooks</h1>
          <Button asChild>
            <Link href={`/${owner}/${repo}/settings/hooks/new`}>
              Add webhook
            </Link>
          </Button>
        </div>

        <p className="mb-6 text-sm text-[#656d76]">
          Webhooks allow external services to be notified when certain events
          happen in this repository.
        </p>

        {loading ? (
          <p className="text-sm text-[#656d76]">Loading…</p>
        ) : error ? (
          <p className="text-sm text-[#cf222e]">{error}</p>
        ) : webhooks.length === 0 ? (
          <div className="rounded-md border border-[#d0d7de] bg-white p-8 text-center">
            <h2 className="text-lg font-semibold">No webhooks configured</h2>
            <p className="mt-2 text-sm text-[#656d76]">
              Add a webhook to receive HTTP POST payloads when events occur in
              this repository.
            </p>
            <Button asChild className="mt-4">
              <Link href={`/${owner}/${repo}/settings/hooks/new`}>
                Add webhook
              </Link>
            </Button>
          </div>
        ) : (
          <div className="overflow-hidden rounded-md border border-[#d0d7de] bg-white">
            <table className="w-full table-auto text-sm">
              <thead className="border-b border-[#d0d7de] bg-[#f6f8fa] text-left text-xs uppercase text-[#656d76]">
                <tr>
                  <th className="px-4 py-2">URL</th>
                  <th className="px-4 py-2">Content-Type</th>
                  <th className="px-4 py-2">Events</th>
                  <th className="px-4 py-2">Active</th>
                  <th className="px-4 py-2">Last Delivery Status</th>
                  <th className="px-4 py-2">Actions</th>
                </tr>
              </thead>
              <tbody>
                {webhooks.map((hook) => (
                  <tr
                    key={hook.id}
                    className="border-b border-[#eaeef2] last:border-b-0"
                  >
                    <td className="max-w-xs truncate px-4 py-2 font-mono text-xs">
                      {hook.config.url}
                    </td>
                    <td className="px-4 py-2">
                      <Badge variant="secondary">
                        {hook.config.content_type === "form"
                          ? "application/x-www-form-urlencoded"
                          : "application/json"}
                      </Badge>
                    </td>
                    <td className="px-4 py-2 text-xs">
                      {formatEvents(hook.events)}
                    </td>
                    <td className="px-4 py-2">
                      <ActiveSwitch active={hook.active} />
                    </td>
                    <td className="px-4 py-2 text-xs">
                      {lastDeliveryLabel(hook)}
                    </td>
                    <td className="px-4 py-2">
                      <div className="flex items-center gap-2">
                        <Button variant="outline" size="sm" asChild>
                          <Link
                            href={`/${owner}/${repo}/settings/hooks/${hook.id}/edit`}
                          >
                            Edit
                          </Link>
                        </Button>
                        {hook.deliveries_count_24h !== undefined && (
                          <Badge variant="secondary">
                            {hook.deliveries_count_24h} in 24h
                          </Badge>
                        )}
                        <Button
                          variant="destructive"
                          size="sm"
                          onClick={() => setDeleteTarget(hook)}
                        >
                          Delete
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {deleteTarget && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
          role="dialog"
          aria-modal="true"
          aria-labelledby="delete-webhook-title"
        >
          <div className="mx-4 w-full max-w-md rounded-md border border-[#d0d7de] bg-white p-6 shadow-lg">
            <h2
              id="delete-webhook-title"
              className="text-lg font-semibold text-[#cf222e]"
            >
              Delete webhook?
            </h2>
            <p className="mt-2 text-sm text-[#656d76]">
              This will permanently remove the webhook for{" "}
              <span className="font-mono">{deleteTarget.config.url}</span>.
              This action cannot be undone.
            </p>
            {deleteError && (
              <p className="mt-2 text-sm text-[#cf222e]">{deleteError}</p>
            )}
            <div className="mt-4 flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => {
                  setDeleteTarget(null);
                  setDeleteError(null);
                }}
                disabled={deleting}
              >
                Cancel
              </Button>
              <Button
                variant="destructive"
                onClick={handleDelete}
                disabled={deleting}
              >
                {deleting ? "Deleting…" : "Delete webhook"}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
