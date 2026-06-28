"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  createRegistrationToken,
  deleteRunner,
} from "@/lib/api/runners";
import type { Runner } from "@/types/runner";

function StatusBadge({ status }: { status: Runner["status"] }) {
  const className =
    status === "online"
      ? "border-transparent bg-[#1f883d] text-white"
      : status === "offline"
        ? "border-transparent bg-[#cf222e] text-white"
        : "border-transparent bg-[#bf8700] text-white";

  return (
    <Badge className={className}>
      {status}
    </Badge>
  );
}

function formatLastSeen(lastSeenAt: string | null): string {
  if (!lastSeenAt) return "—";
  return new Date(lastSeenAt).toLocaleString();
}

export function RegistrationTokenModal({
  org,
  open,
  onClose,
}: {
  org: string;
  open: boolean;
  onClose: () => void;
}) {
  const [token, setToken] = useState<string | null>(null);
  const [expiresAt, setExpiresAt] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const handleCreate = async () => {
    setLoading(true);
    setError(null);
    setCopied(false);
    try {
      const response = await createRegistrationToken(org);
      setToken(response.token);
      setExpiresAt(response.expires_at);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to create registration token.",
      );
    } finally {
      setLoading(false);
    }
  };

  const handleCopy = async () => {
    if (!token) return;
    await navigator.clipboard.writeText(token);
    setCopied(true);
  };

  const handleClose = () => {
    setToken(null);
    setExpiresAt(null);
    setError(null);
    setCopied(false);
    onClose();
  };

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      role="dialog"
      aria-modal="true"
      aria-labelledby="registration-token-title"
    >
      <div className="mx-4 w-full max-w-lg rounded-md border border-[#d0d7de] bg-white p-6 shadow-lg">
        <h2 id="registration-token-title" className="text-lg font-semibold">
          New registration token
        </h2>
        <p className="mt-2 text-sm text-[#656d76]">
          This token is shown only once. Copy it now and use it to register a
          self-hosted runner before it expires.
        </p>

        {error ? (
          <p className="mt-3 text-sm text-[#cf222e]">{error}</p>
        ) : null}

        {token ? (
          <div className="mt-4 space-y-3">
            <div className="flex gap-2">
              <Input readOnly value={token} className="font-mono text-xs" />
              <Button type="button" variant="outline" onClick={handleCopy}>
                {copied ? "Copied" : "Copy"}
              </Button>
            </div>
            {expiresAt ? (
              <p className="text-sm text-[#bf8700]">
                Expires at {new Date(expiresAt).toLocaleString()}
              </p>
            ) : null}
          </div>
        ) : (
          <Button
            type="button"
            className="mt-4"
            onClick={handleCreate}
            disabled={loading}
          >
            {loading ? "Generating…" : "Generate token"}
          </Button>
        )}

        <div className="mt-4 flex justify-end">
          <Button type="button" variant="outline" onClick={handleClose}>
            Close
          </Button>
        </div>
      </div>
    </div>
  );
}

export function RunnersPageClient({
  org,
  initialRunners,
}: {
  org: string;
  initialRunners: Runner[];
}) {
  const router = useRouter();
  const [modalOpen, setModalOpen] = useState(false);
  const [runners, setRunners] = useState(initialRunners);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  useEffect(() => {
    setRunners(initialRunners);
  }, [initialRunners]);

  const handleDelete = async (runnerId: string) => {
    setDeletingId(runnerId);
    setDeleteError(null);
    try {
      await deleteRunner(org, runnerId);
      setRunners((current) => current.filter((runner) => runner.id !== runnerId));
      router.refresh();
    } catch (err) {
      setDeleteError(
        err instanceof Error ? err.message : "Failed to delete runner.",
      );
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <>
      <div className="mb-4 flex items-center justify-between">
        <p className="text-sm text-[#656d76]">
          Manage self-hosted runners for your organization.
        </p>
        <Button type="button" onClick={() => setModalOpen(true)}>
          New registration token
        </Button>
      </div>

      {deleteError ? (
        <p className="mb-4 text-sm text-[#cf222e]">{deleteError}</p>
      ) : null}

      {runners.length === 0 ? (
        <div className="rounded-md border border-[#d0d7de] bg-white p-8 text-center text-sm text-[#656d76]">
          No runners registered
        </div>
      ) : (
        <div className="overflow-hidden rounded-md border border-[#d0d7de] bg-white">
          <table className="w-full table-auto text-sm">
            <thead className="border-b border-[#d0d7de] bg-[#f6f8fa] text-left text-xs uppercase text-[#656d76]">
              <tr>
                <th className="px-4 py-2">Name</th>
                <th className="px-4 py-2">Status</th>
                <th className="px-4 py-2">Labels</th>
                <th className="px-4 py-2">Type</th>
                <th className="px-4 py-2">Last Seen</th>
                <th className="px-4 py-2">Actions</th>
              </tr>
            </thead>
            <tbody>
              {runners.map((runner) => (
                <tr
                  key={runner.id}
                  className="border-b border-[#eaeef2] last:border-b-0"
                >
                  <td className="px-4 py-2">{runner.name}</td>
                  <td className="px-4 py-2">
                    <StatusBadge status={runner.status} />
                  </td>
                  <td className="px-4 py-2 text-xs">
                    {runner.labels.length > 0 ? runner.labels.join(", ") : "—"}
                  </td>
                  <td className="px-4 py-2">{runner.runner_type}</td>
                  <td className="px-4 py-2 text-xs">
                    {formatLastSeen(runner.last_seen_at)}
                  </td>
                  <td className="px-4 py-2">
                    <Button
                      type="button"
                      variant="destructive"
                      size="sm"
                      disabled={deletingId === runner.id}
                      onClick={() => handleDelete(runner.id)}
                    >
                      {deletingId === runner.id ? "Deleting…" : "Delete"}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <RegistrationTokenModal
        org={org}
        open={modalOpen}
        onClose={() => setModalOpen(false)}
      />
    </>
  );
}
