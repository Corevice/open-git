"use client";

import { use, useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { SecretForm } from "@/components/secret-form";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api";
import {
  deleteOrgSecret,
  getOrgPublicKey,
  isSecretValidationError,
  listOrgSecrets,
  sealSecret,
  upsertOrgSecret,
  validateSecretName,
  type OrgActionSecret,
  type SecretVisibility,
} from "@/lib/api/secrets";

function formatDate(value: string | undefined): string {
  if (!value) {
    return "—";
  }
  return new Date(value).toLocaleString();
}

function visibilityLabel(visibility: SecretVisibility): string {
  switch (visibility) {
    case "all":
      return "All repositories";
    case "private":
      return "Private repositories";
    case "selected":
      return "Selected repositories";
  }
}

function VisibilityBadge({ visibility }: { visibility: SecretVisibility }) {
  const variant =
    visibility === "all"
      ? "default"
      : visibility === "private"
        ? "secondary"
        : "outline";

  return <Badge variant={variant}>{visibilityLabel(visibility)}</Badge>;
}

export default function OrganizationSecretsPage({
  params,
}: {
  params: Promise<{ owner: string }>;
}) {
  const { owner } = use(params);
  const router = useRouter();
  const loadIdRef = useRef(0);

  const [secrets, setSecrets] = useState<OrgActionSecret[]>([]);
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

  const editingSecretData = secrets.find(
    (secret) => secret.name === editingSecret,
  );

  const redirectOnUnauthorized = useCallback(
    (err: unknown): boolean => {
      if (err instanceof ApiError && err.status === 401) {
        router.push("/login");
        return true;
      }
      return false;
    },
    [router],
  );

  const loadSecrets = useCallback(async () => {
    const loadId = ++loadIdRef.current;
    setLoading(true);
    setError(null);
    try {
      const data = await listOrgSecrets(owner);
      if (loadId !== loadIdRef.current) {
        return;
      }
      setSecrets(data);
    } catch (err) {
      if (redirectOnUnauthorized(err)) {
        return;
      }
      if (loadId !== loadIdRef.current) {
        return;
      }
      setError(
        err instanceof Error ? err.message : "Failed to load organization secrets.",
      );
    } finally {
      if (loadId === loadIdRef.current) {
        setLoading(false);
      }
    }
  }, [owner, redirectOnUnauthorized]);

  useEffect(() => {
    void loadSecrets();
  }, [loadSecrets]);

  const resetFormState = () => {
    setFormError(null);
    setFieldErrors({});
  };

  const openCreateForm = () => {
    resetFormState();
    setDeleteError(null);
    setShowCreateForm(true);
    setEditingSecret(null);
  };

  const openEditForm = (secret: OrgActionSecret) => {
    resetFormState();
    setDeleteError(null);
    setEditingSecret(secret.name);
    setShowCreateForm(false);
  };

  const handleUpsert = async (
    name: string,
    value: string,
    visibility: SecretVisibility = "all",
  ) => {
    setSubmitting(true);
    setFormError(null);
    setFieldErrors({});

    const isUpdate = editingSecret !== null;
    const nameError = isUpdate ? null : validateSecretName(name);
    if (nameError) {
      setFieldErrors({ name: nameError });
      setFormError(nameError);
      setSubmitting(false);
      return;
    }

    const trimmedValue = value.trim();
    if (!isUpdate && !trimmedValue) {
      setFieldErrors({ value: "Secret value is required" });
      setFormError("Secret value is required");
      setSubmitting(false);
      return;
    }

    try {
      if (trimmedValue) {
        const publicKey = await getOrgPublicKey(owner);
        const { encrypted_value, key_id } = await sealSecret(
          trimmedValue,
          publicKey,
        );
        await upsertOrgSecret(
          owner,
          name,
          encrypted_value,
          key_id,
          visibility,
        );
      } else {
        await upsertOrgSecret(
          owner,
          name,
          undefined,
          undefined,
          visibility,
        );
      }

      setShowCreateForm(false);
      setEditingSecret(null);
      resetFormState();
      await loadSecrets();
    } catch (err) {
      if (redirectOnUnauthorized(err)) {
        return;
      }
      if (isSecretValidationError(err)) {
        setFieldErrors(err.fieldErrors);
        setFormError(err.message);
      } else {
        setFormError(
          err instanceof Error
            ? err.message
            : isUpdate
              ? "Failed to update secret."
              : "Failed to create secret.",
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
      await deleteOrgSecret(owner, name);
      setSecrets((prev) => prev.filter((secret) => secret.name !== name));
      setDeletingSecret(null);
    } catch (err) {
      if (redirectOnUnauthorized(err)) {
        return;
      }
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
          href={`/${owner}/settings`}
          className="rounded-md px-3 py-1.5 text-sm hover:bg-[#f6f8fa]"
        >
          ← Settings
        </Link>
      </header>

      <div className="mx-auto max-w-[1200px] px-6 py-6">
        <div className="mb-4 text-sm text-[#656d76]">
          <Link href={`/${owner}`} className="text-[#0969da]">
            {owner}
          </Link>{" "}
          /{" "}
          <Link href={`/${owner}/settings`} className="text-[#0969da]">
            Settings
          </Link>{" "}
          / Secrets
        </div>

        <div className="mb-4 flex items-center justify-between">
          <h1 className="text-2xl font-semibold">Organization secrets</h1>
          <Button onClick={openCreateForm}>New organization secret</Button>
        </div>

        <p className="mb-6 text-sm text-[#656d76]">
          Organization secrets are encrypted environment variables available to
          GitHub Actions workflows across repositories in this organization.
        </p>

        {showCreateForm && editingSecret === null && (
          <div className="mb-6 rounded-md border border-[#d0d7de] bg-white p-6">
            <h2 className="mb-4 text-lg font-semibold">
              New organization secret
            </h2>
            <SecretForm
              onSubmit={handleUpsert}
              showVisibility
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
          <p className="text-sm text-destructive">{error}</p>
        ) : secrets.length === 0 ? (
          <div className="rounded-md border border-[#d0d7de] bg-white p-8 text-center">
            <h2 className="text-lg font-semibold">No secrets configured</h2>
            <p className="mt-2 text-sm text-[#656d76]">
              Add a secret to make it available to workflows in this
              organization.
            </p>
            <Button className="mt-4" onClick={openCreateForm}>
              New organization secret
            </Button>
          </div>
        ) : (
          <div className="overflow-hidden rounded-md border border-[#d0d7de] bg-white">
            <table className="w-full table-auto text-sm">
              <thead className="border-b border-[#d0d7de] bg-[#f6f8fa] text-left text-xs uppercase text-[#656d76]">
                <tr>
                  <th className="px-4 py-2">Name</th>
                  <th className="px-4 py-2">Visibility</th>
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
                    <td className="px-4 py-2 font-mono text-xs">
                      {secret.name}
                    </td>
                    <td className="px-4 py-2">
                      <VisibilityBadge visibility={secret.visibility} />
                    </td>
                    <td className="px-4 py-2 text-xs">
                      {formatDate(secret.updated_at)}
                    </td>
                    <td className="px-4 py-2">
                      <div className="flex items-center gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => openEditForm(secret)}
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

        {editingSecret !== null && !showCreateForm && editingSecretData && (
          <div className="mt-6 rounded-md border border-[#d0d7de] bg-white p-6">
            <h2 className="mb-4 text-lg font-semibold">
              Update secret: {editingSecret}
            </h2>
            <SecretForm
              onSubmit={handleUpsert}
              existingName={editingSecret}
              initialVisibility={editingSecretData.visibility}
              showVisibility
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
              className="text-lg font-semibold text-destructive"
            >
              Delete secret?
            </h2>
            <p className="mt-2 text-sm text-[#656d76]">
              This will permanently remove the secret{" "}
              <span className="font-mono">{deletingSecret}</span>. This action
              cannot be undone.
            </p>
            {deleteError && (
              <p className="mt-2 text-sm text-destructive">{deleteError}</p>
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
