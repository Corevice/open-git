"use client";

import { use, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { SecretForm } from "@/components/secret-form";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api";
import {
  deleteRepoSecret,
  getRepoPublicKey,
  isSecretValidationError,
  listRepoSecrets,
  sealSecret,
  upsertRepoSecret,
  type ActionSecret,
} from "@/lib/api/secrets";

function formatDate(value: string | undefined): string {
  if (!value) {
    return "—";
  }
  return new Date(value).toLocaleString();
}

export default function RepositorySecretsPage({
  params,
}: {
  params: Promise<{ owner: string; repo: string }>;
}) {
  const { owner, repo } = use(params);
  const router = useRouter();

  const [secrets, setSecrets] = useState<ActionSecret[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [editingSecret, setEditingSecret] = useState<string | null>(null);
  const [deletingSecret, setDeletingSecret] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);

  const loadSecrets = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await listRepoSecrets(owner, repo);
      setSecrets(data);
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        router.push("/login");
        return;
      }
      setError(
        err instanceof Error ? err.message : "Failed to load repository secrets.",
      );
    } finally {
      setLoading(false);
    }
  }, [owner, repo, router]);

  useEffect(() => {
    void loadSecrets();
  }, [loadSecrets]);

  const resetFormState = () => {
    setFormError(null);
    setFieldErrors({});
  };

  const handleCreate = async (name: string, value: string) => {
    setSubmitting(true);
    resetFormState();
    try {
      const publicKey = await getRepoPublicKey(owner, repo);
      const { encrypted_value, key_id } = await sealSecret(value, publicKey);
      await upsertRepoSecret(owner, repo, name, encrypted_value, key_id);
      setShowCreateForm(false);
      await loadSecrets();
    } catch (err) {
      if (isSecretValidationError(err)) {
        setFieldErrors(err.fieldErrors);
        setFormError(err.message);
      } else {
        setFormError(
          err instanceof Error ? err.message : "Failed to create secret.",
        );
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleUpdate = async (name: string, value: string) => {
    setSubmitting(true);
    resetFormState();
    try {
      const publicKey = await getRepoPublicKey(owner, repo);
      const { encrypted_value, key_id } = await sealSecret(value, publicKey);
      await upsertRepoSecret(owner, repo, name, encrypted_value, key_id);
      setEditingSecret(null);
      await loadSecrets();
    } catch (err) {
      if (isSecretValidationError(err)) {
        setFieldErrors(err.fieldErrors);
        setFormError(err.message);
      } else {
        setFormError(
          err instanceof Error ? err.message : "Failed to update secret.",
        );
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (name: string) => {
    setDeleting(true);
    setDeleteError(null);
    try {
      await deleteRepoSecret(owner, repo, name);
      setSecrets((prev) => prev.filter((secret) => secret.name !== name));
      setDeletingSecret(null);
    } catch (err) {
      setDeleteError(
        err instanceof Error ? err.message : "Failed to delete secret.",
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
          / Secrets
        </div>

        <div className="mb-4 flex items-center justify-between">
          <h1 className="text-2xl font-semibold">Repository secrets</h1>
          <Button
            onClick={() => {
              resetFormState();
              setShowCreateForm(true);
              setEditingSecret(null);
            }}
          >
            New repository secret
          </Button>
        </div>

        <p className="mb-6 text-sm text-[#656d76]">
          Secrets are encrypted environment variables available to GitHub
          Actions workflows in this repository.
        </p>

        {showCreateForm && (
          <div className="mb-6 rounded-md border border-[#d0d7de] bg-white p-6">
            <h2 className="mb-4 text-lg font-semibold">New repository secret</h2>
            <SecretForm
              onSubmit={handleCreate}
              submitting={submitting}
              formError={formError}
              fieldErrors={fieldErrors}
              onCancel={() => {
                setShowCreateForm(false);
                resetFormState();
              }}
            />
          </div>
        )}

        {loading ? (
          <p className="text-sm text-[#656d76]">Loading…</p>
        ) : error ? (
          <p className="text-sm text-[#cf222e]">{error}</p>
        ) : secrets.length === 0 ? (
          <div className="rounded-md border border-[#d0d7de] bg-white p-8 text-center">
            <h2 className="text-lg font-semibold">No secrets configured</h2>
            <p className="mt-2 text-sm text-[#656d76]">
              Add a secret to make it available to workflows in this repository.
            </p>
            <Button
              className="mt-4"
              onClick={() => {
                resetFormState();
                setShowCreateForm(true);
              }}
            >
              New repository secret
            </Button>
          </div>
        ) : (
          <div className="overflow-hidden rounded-md border border-[#d0d7de] bg-white">
            <table className="w-full table-auto text-sm">
              <thead className="border-b border-[#d0d7de] bg-[#f6f8fa] text-left text-xs uppercase text-[#656d76]">
                <tr>
                  <th className="px-4 py-2">Name</th>
                  <th className="px-4 py-2">Updated</th>
                  <th className="px-4 py-2">Actions</th>
                </tr>
              </thead>
              <tbody>
                {secrets.map((secret) => (
                  <tr
                    key={secret.name}
                    className="border-b border-[#eaeef2] last:border-b-0"
                  >
                    <td className="px-4 py-2 font-mono text-xs">{secret.name}</td>
                    <td className="px-4 py-2 text-xs">
                      {formatDate(secret.updated_at)}
                    </td>
                    <td className="px-4 py-2">
                      <div className="flex items-center gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => {
                            resetFormState();
                            setEditingSecret(secret.name);
                            setShowCreateForm(false);
                          }}
                        >
                          Update
                        </Button>
                        <Button
                          variant="destructive"
                          size="sm"
                          onClick={() => {
                            setDeleteError(null);
                            setDeletingSecret(secret.name);
                          }}
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

        {editingSecret !== null && (
          <div className="mt-6 rounded-md border border-[#d0d7de] bg-white p-6">
            <h2 className="mb-4 text-lg font-semibold">
              Update secret: {editingSecret}
            </h2>
            <SecretForm
              existingName={editingSecret}
              onSubmit={handleUpdate}
              submitting={submitting}
              formError={formError}
              fieldErrors={fieldErrors}
              onCancel={() => {
                setEditingSecret(null);
                resetFormState();
              }}
            />
          </div>
        )}
      </div>

      {deletingSecret && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
          role="dialog"
          aria-modal="true"
          aria-labelledby="delete-secret-title"
        >
          <div className="mx-4 w-full max-w-md rounded-md border border-[#d0d7de] bg-white p-6 shadow-lg">
            <h2
              id="delete-secret-title"
              className="text-lg font-semibold text-[#cf222e]"
            >
              Delete secret?
            </h2>
            <p className="mt-2 text-sm text-[#656d76]">
              This will permanently remove the secret{" "}
              <span className="font-mono">{deletingSecret}</span>. This action
              cannot be undone.
            </p>
            {deleteError && (
              <p className="mt-2 text-sm text-[#cf222e]">{deleteError}</p>
            )}
            <div className="mt-4 flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => {
                  setDeletingSecret(null);
                  setDeleteError(null);
                }}
                disabled={deleting}
              >
                Cancel
              </Button>
              <Button
                variant="destructive"
                onClick={() => void handleDelete(deletingSecret)}
                disabled={deleting}
              >
                {deleting ? "Deleting…" : "Delete secret"}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
