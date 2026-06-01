"use client";

import { FormEvent, use, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import BranchProtectionForm, {
  type BranchProtectionInitial,
} from "@/components/repo/BranchProtectionForm";
import WebhookForm, {
  type WebhookInitial,
} from "@/components/repo/WebhookForm";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";

type Tab = "general" | "branches" | "webhooks" | "secrets" | "danger";

interface RepoMetadata {
  name: string;
  description: string | null;
  private: boolean;
  visibility?: "public" | "private" | "internal";
  default_branch: string;
  owner: { login: string };
}

interface BranchProtection {
  pattern: string;
  required_reviews: number;
  required_checks: string[];
  force_push_blocked: boolean;
}

interface Webhook {
  id: number;
  config?: { url?: string };
  url?: string;
  events: string[];
  active: boolean;
}

interface SecretEntry {
  name: string;
  updated_at?: string;
}

async function jsonFetch<T>(
  path: string,
  init?: RequestInit,
): Promise<T | null> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      Accept: "application/vnd.github+json",
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...(init?.headers ?? {}),
    },
    cache: "no-store",
  });
  if (res.status === 404) return null;
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(
      (body as { message?: string }).message ??
        `Request failed (${res.status})`,
    );
  }
  if (res.status === 204) return null;
  return (await res.json()) as T;
}

export default function RepoSettingsPage({
  params,
}: {
  params: Promise<{ owner: string; repo: string }>;
}) {
  const { owner, repo } = use(params);
  const router = useRouter();

  const [tab, setTab] = useState<Tab>("general");
  const [meta, setMeta] = useState<RepoMetadata | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  const reloadMeta = useCallback(async () => {
    try {
      const data = await jsonFetch<RepoMetadata>(`/repos/${owner}/${repo}`);
      setMeta(data);
      setLoadError(null);
    } catch (err) {
      setLoadError(
        err instanceof Error ? err.message : "Failed to load repository.",
      );
    }
  }, [owner, repo]);

  useEffect(() => {
    reloadMeta();
  }, [reloadMeta]);

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
          href={`/${owner}/${repo}`}
          className="rounded-md px-3 py-1.5 text-sm hover:bg-[#f6f8fa]"
        >
          ← Back to repository
        </Link>
      </header>

      <div className="mx-auto max-w-[1200px] px-6 py-6">
        <div className="mb-4 text-sm text-[#656d76]">
          <Link href={`/${owner}/${repo}`} className="text-[#0969da]">
            {owner}/{repo}
          </Link>{" "}
          / Settings
        </div>
        <h1 className="mb-4 text-2xl font-semibold">Settings</h1>

        {loadError && (
          <p className="mb-4 rounded-md border border-[#cf222e] bg-[#ffebe9] p-3 text-sm text-[#cf222e]">
            {loadError}
          </p>
        )}

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-[220px_1fr]">
          <aside>
            <nav className="rounded-md border border-[#d0d7de] bg-white p-2">
              <TabButton
                current={tab}
                value="general"
                onClick={() => setTab("general")}
                label="General"
              />
              <TabButton
                current={tab}
                value="branches"
                onClick={() => setTab("branches")}
                label="Branches"
              />
              <TabButton
                current={tab}
                value="webhooks"
                onClick={() => setTab("webhooks")}
                label="Webhooks"
              />
              <TabButton
                current={tab}
                value="secrets"
                onClick={() => setTab("secrets")}
                label="Secrets"
              />
              <TabButton
                current={tab}
                value="danger"
                onClick={() => setTab("danger")}
                label="Danger Zone"
                danger
              />
              <Link
                href={`/${owner}/${repo}/settings/audit`}
                className="mt-2 block rounded-md px-3 py-2 text-sm text-[#0969da] hover:bg-[#f6f8fa]"
              >
                Audit log →
              </Link>
            </nav>
          </aside>

          <main className="space-y-6">
            {tab === "general" && meta && (
              <GeneralTab
                owner={owner}
                repo={repo}
                meta={meta}
                onSaved={reloadMeta}
              />
            )}
            {tab === "branches" && (
              <BranchesTab owner={owner} repo={repo} />
            )}
            {tab === "webhooks" && (
              <WebhooksTab owner={owner} repo={repo} />
            )}
            {tab === "secrets" && (
              <SecretsTab owner={owner} repo={repo} />
            )}
            {tab === "danger" && meta && (
              <DangerTab
                owner={owner}
                repo={repo}
                meta={meta}
                onVisibilityChanged={reloadMeta}
                onDeleted={() => router.push("/dashboard")}
              />
            )}
          </main>
        </div>
      </div>
    </div>
  );
}

function TabButton({
  current,
  value,
  onClick,
  label,
  danger,
}: {
  current: Tab;
  value: Tab;
  onClick: () => void;
  label: string;
  danger?: boolean;
}) {
  const active = current === value;
  return (
    <button
      type="button"
      onClick={onClick}
      className={`block w-full rounded-md px-3 py-2 text-left text-sm ${
        active
          ? "bg-[#ddf4ff] font-semibold text-[#0969da]"
          : danger
            ? "text-[#cf222e] hover:bg-[#ffebe9]"
            : "text-[#1f2328] hover:bg-[#f6f8fa]"
      }`}
    >
      {label}
    </button>
  );
}

function GeneralTab({
  owner,
  repo,
  meta,
  onSaved,
}: {
  owner: string;
  repo: string;
  meta: RepoMetadata;
  onSaved: () => void;
}) {
  const [name, setName] = useState(meta.name);
  const [description, setDescription] = useState(meta.description ?? "");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    setName(meta.name);
    setDescription(meta.description ?? "");
  }, [meta]);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    setSubmitting(true);
    try {
      await jsonFetch(`/repos/${owner}/${repo}`, {
        method: "PATCH",
        body: JSON.stringify({
          name: name.trim(),
          description: description.trim(),
        }),
      });
      setSuccess("Repository updated.");
      onSaved();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="space-y-4 rounded-md border border-[#d0d7de] bg-white p-5"
    >
      <h2 className="text-lg font-semibold">General</h2>
      <div>
        <label
          htmlFor="general-name"
          className="mb-1.5 block text-sm font-semibold"
        >
          Repository name <span className="text-[#cf222e]">*</span>
        </label>
        <input
          id="general-name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm"
          required
          pattern="[a-zA-Z0-9._-]{1,100}"
        />
      </div>
      <div>
        <label
          htmlFor="general-description"
          className="mb-1.5 block text-sm font-semibold"
        >
          Description
        </label>
        <textarea
          id="general-description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={3}
          className="w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm"
        />
      </div>
      {error && <p className="text-sm text-[#cf222e]">{error}</p>}
      {success && <p className="text-sm text-[#1f883d]">{success}</p>}
      <div className="flex justify-end">
        <button
          type="submit"
          disabled={submitting}
          className="rounded-md bg-[#1f883d] px-4 py-2 text-sm font-semibold text-white hover:bg-[#1a7f37] disabled:opacity-50"
        >
          {submitting ? "Saving…" : "Save changes"}
        </button>
      </div>
    </form>
  );
}

function BranchesTab({ owner, repo }: { owner: string; repo: string }) {
  const [rules, setRules] = useState<BranchProtection[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState<BranchProtectionInitial | null>(null);
  const [showCreate, setShowCreate] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await jsonFetch<BranchProtection[]>(
        `/repos/${owner}/${repo}/branches/protection`,
      );
      setRules(data ?? []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load rules.");
    } finally {
      setLoading(false);
    }
  }, [owner, repo]);

  useEffect(() => {
    load();
  }, [load]);

  const handleSaved = () => {
    setShowCreate(false);
    setEditing(null);
    load();
  };

  const handleDelete = async (pattern: string) => {
    if (!confirm(`Delete protection rule for "${pattern}"?`)) return;
    try {
      await jsonFetch(
        `/repos/${owner}/${repo}/branches/${encodeURIComponent(pattern)}/protection`,
        { method: "DELETE" },
      );
      load();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to delete rule.");
    }
  };

  return (
    <div className="space-y-4">
      <div className="rounded-md border border-[#d0d7de] bg-white p-5">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Branch protection rules</h2>
          {!showCreate && !editing && (
            <button
              type="button"
              onClick={() => setShowCreate(true)}
              className="rounded-md border border-[#d0d7de] bg-[#f6f8fa] px-3 py-1.5 text-sm hover:bg-white"
            >
              + Add rule
            </button>
          )}
        </div>
        {loading ? (
          <p className="mt-3 text-sm text-[#656d76]">Loading…</p>
        ) : error ? (
          <p className="mt-3 text-sm text-[#cf222e]">{error}</p>
        ) : rules.length === 0 ? (
          <p className="mt-3 text-sm text-[#656d76]">
            No branch protection rules configured.
          </p>
        ) : (
          <ul className="mt-3 divide-y divide-[#eaeef2]">
            {rules.map((rule) => (
              <li
                key={rule.pattern}
                className="flex items-center justify-between py-3"
              >
                <div>
                  <div className="font-mono text-sm font-semibold">
                    {rule.pattern}
                  </div>
                  <div className="mt-0.5 text-xs text-[#656d76]">
                    {rule.required_reviews} approving review
                    {rule.required_reviews === 1 ? "" : "s"} ·{" "}
                    {rule.required_checks?.length ?? 0} required check
                    {(rule.required_checks?.length ?? 0) === 1 ? "" : "s"} ·{" "}
                    Force push{" "}
                    {rule.force_push_blocked ? "blocked" : "allowed"}
                  </div>
                </div>
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={() => setEditing(rule)}
                    className="rounded-md border border-[#d0d7de] bg-white px-3 py-1.5 text-xs hover:bg-[#f6f8fa]"
                  >
                    Edit
                  </button>
                  <button
                    type="button"
                    onClick={() => handleDelete(rule.pattern)}
                    className="rounded-md border border-[#d0d7de] bg-white px-3 py-1.5 text-xs text-[#cf222e] hover:bg-[#ffebe9]"
                  >
                    Delete
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>

      {showCreate && (
        <BranchProtectionForm
          owner={owner}
          repo={repo}
          mode="create"
          onSaved={handleSaved}
        />
      )}
      {editing && (
        <BranchProtectionForm
          owner={owner}
          repo={repo}
          initial={editing}
          mode="edit"
          onSaved={handleSaved}
        />
      )}
    </div>
  );
}

function WebhooksTab({ owner, repo }: { owner: string; repo: string }) {
  const [hooks, setHooks] = useState<Webhook[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState<WebhookInitial | null>(null);
  const [showCreate, setShowCreate] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await jsonFetch<Webhook[]>(
        `/repos/${owner}/${repo}/hooks`,
      );
      setHooks(data ?? []);
      setError(null);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load webhooks.",
      );
    } finally {
      setLoading(false);
    }
  }, [owner, repo]);

  useEffect(() => {
    load();
  }, [load]);

  const handleSaved = () => {
    setShowCreate(false);
    setEditing(null);
    load();
  };

  const handleDelete = async (id: number) => {
    if (!confirm("Delete this webhook?")) return;
    try {
      await jsonFetch(`/repos/${owner}/${repo}/hooks/${id}`, {
        method: "DELETE",
      });
      load();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to delete webhook.");
    }
  };

  return (
    <div className="space-y-4">
      <div className="rounded-md border border-[#d0d7de] bg-white p-5">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Webhooks</h2>
          {!showCreate && !editing && (
            <button
              type="button"
              onClick={() => setShowCreate(true)}
              className="rounded-md border border-[#d0d7de] bg-[#f6f8fa] px-3 py-1.5 text-sm hover:bg-white"
            >
              + Add webhook
            </button>
          )}
        </div>
        {loading ? (
          <p className="mt-3 text-sm text-[#656d76]">Loading…</p>
        ) : error ? (
          <p className="mt-3 text-sm text-[#cf222e]">{error}</p>
        ) : hooks.length === 0 ? (
          <p className="mt-3 text-sm text-[#656d76]">
            No webhooks configured.
          </p>
        ) : (
          <ul className="mt-3 divide-y divide-[#eaeef2]">
            {hooks.map((hook) => {
              const hookUrl = hook.config?.url ?? hook.url ?? "(no url)";
              return (
                <li
                  key={hook.id}
                  className="flex items-center justify-between py-3"
                >
                  <div className="min-w-0 flex-1">
                    <div className="truncate font-mono text-sm font-semibold">
                      {hookUrl}
                    </div>
                    <div className="mt-0.5 text-xs text-[#656d76]">
                      Events: {hook.events.join(", ") || "(none)"} ·{" "}
                      {hook.active ? "Active" : "Inactive"}
                    </div>
                  </div>
                  <div className="ml-3 flex gap-2">
                    <button
                      type="button"
                      onClick={() =>
                        setEditing({
                          id: hook.id,
                          url: hookUrl,
                          events: hook.events,
                          active: hook.active,
                        })
                      }
                      className="rounded-md border border-[#d0d7de] bg-white px-3 py-1.5 text-xs hover:bg-[#f6f8fa]"
                    >
                      Edit
                    </button>
                    <button
                      type="button"
                      onClick={() => handleDelete(hook.id)}
                      className="rounded-md border border-[#d0d7de] bg-white px-3 py-1.5 text-xs text-[#cf222e] hover:bg-[#ffebe9]"
                    >
                      Delete
                    </button>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </div>

      {showCreate && (
        <WebhookForm
          owner={owner}
          repo={repo}
          mode="create"
          onSaved={handleSaved}
        />
      )}
      {editing && (
        <WebhookForm
          owner={owner}
          repo={repo}
          initial={editing}
          mode="edit"
          onSaved={handleSaved}
        />
      )}
    </div>
  );
}

function SecretsTab({ owner, repo }: { owner: string; repo: string }) {
  const [secrets, setSecrets] = useState<SecretEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [value, setValue] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [formSuccess, setFormSuccess] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await jsonFetch<{ secrets: SecretEntry[] } | SecretEntry[]>(
        `/repos/${owner}/${repo}/actions/secrets`,
      );
      if (Array.isArray(data)) {
        setSecrets(data);
      } else {
        setSecrets(data?.secrets ?? []);
      }
      setError(null);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load secrets.",
      );
    } finally {
      setLoading(false);
    }
  }, [owner, repo]);

  useEffect(() => {
    load();
  }, [load]);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setFormError(null);
    setFormSuccess(null);
    if (!name.trim()) {
      setFormError("Secret name is required.");
      return;
    }
    if (!value) {
      setFormError("Secret value is required.");
      return;
    }
    setSubmitting(true);
    try {
      await jsonFetch(
        `/repos/${owner}/${repo}/actions/secrets/${encodeURIComponent(name.trim())}`,
        {
          method: "PUT",
          body: JSON.stringify({ encrypted_value: value }),
        },
      );
      setFormSuccess(`Secret "${name.trim()}" saved.`);
      setName("");
      setValue("");
      load();
    } catch (err) {
      setFormError(
        err instanceof Error ? err.message : "Failed to save secret.",
      );
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (secretName: string) => {
    if (!confirm(`Delete secret "${secretName}"?`)) return;
    try {
      await jsonFetch(
        `/repos/${owner}/${repo}/actions/secrets/${encodeURIComponent(secretName)}`,
        { method: "DELETE" },
      );
      load();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to delete secret.");
    }
  };

  return (
    <div className="space-y-4">
      <div className="rounded-md border border-[#d0d7de] bg-white p-5">
        <h2 className="text-lg font-semibold">Repository secrets</h2>
        <p className="mt-1 text-sm text-[#656d76]">
          Secret values are never displayed after they are stored.
        </p>
        {loading ? (
          <p className="mt-3 text-sm text-[#656d76]">Loading…</p>
        ) : error ? (
          <p className="mt-3 text-sm text-[#cf222e]">{error}</p>
        ) : secrets.length === 0 ? (
          <p className="mt-3 text-sm text-[#656d76]">No secrets configured.</p>
        ) : (
          <ul className="mt-3 divide-y divide-[#eaeef2]">
            {secrets.map((s) => (
              <li
                key={s.name}
                className="flex items-center justify-between py-3"
              >
                <div>
                  <div className="font-mono text-sm font-semibold">
                    {s.name}
                  </div>
                  {s.updated_at && (
                    <div className="mt-0.5 text-xs text-[#656d76]">
                      Updated {s.updated_at}
                    </div>
                  )}
                </div>
                <button
                  type="button"
                  onClick={() => handleDelete(s.name)}
                  className="rounded-md border border-[#d0d7de] bg-white px-3 py-1.5 text-xs text-[#cf222e] hover:bg-[#ffebe9]"
                >
                  Delete
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      <form
        onSubmit={handleSubmit}
        className="space-y-3 rounded-md border border-[#d0d7de] bg-white p-5"
      >
        <h3 className="text-base font-semibold">Add or update a secret</h3>
        <div>
          <label
            htmlFor="secret-name"
            className="mb-1.5 block text-sm font-semibold"
          >
            Name <span className="text-[#cf222e]">*</span>
          </label>
          <input
            id="secret-name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value.toUpperCase())}
            placeholder="MY_SECRET"
            className="w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm font-mono"
            required
          />
        </div>
        <div>
          <label
            htmlFor="secret-value"
            className="mb-1.5 block text-sm font-semibold"
          >
            Value <span className="text-[#cf222e]">*</span>
          </label>
          <textarea
            id="secret-value"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            rows={3}
            className="w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm font-mono"
            required
          />
        </div>
        {formError && <p className="text-sm text-[#cf222e]">{formError}</p>}
        {formSuccess && <p className="text-sm text-[#1f883d]">{formSuccess}</p>}
        <div className="flex justify-end">
          <button
            type="submit"
            disabled={submitting}
            className="rounded-md bg-[#1f883d] px-4 py-2 text-sm font-semibold text-white hover:bg-[#1a7f37] disabled:opacity-50"
          >
            {submitting ? "Saving…" : "Save secret"}
          </button>
        </div>
      </form>
    </div>
  );
}

function DangerTab({
  owner,
  repo,
  meta,
  onVisibilityChanged,
  onDeleted,
}: {
  owner: string;
  repo: string;
  meta: RepoMetadata;
  onVisibilityChanged: () => void;
  onDeleted: () => void;
}) {
  const [showVisibility, setShowVisibility] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  return (
    <div className="space-y-4">
      <div className="rounded-md border border-[#cf222e] bg-white p-5">
        <h2 className="text-lg font-semibold text-[#cf222e]">Danger Zone</h2>

        <div className="mt-4 flex items-start justify-between gap-4 border-t border-[#eaeef2] pt-4">
          <div>
            <div className="text-sm font-semibold">Change visibility</div>
            <p className="text-xs text-[#656d76]">
              This repository is currently{" "}
              <strong>
                {meta.visibility ?? (meta.private ? "private" : "public")}
              </strong>
              .
            </p>
          </div>
          <button
            type="button"
            onClick={() => setShowVisibility(true)}
            className="rounded-md border border-[#cf222e] bg-white px-3 py-1.5 text-sm font-semibold text-[#cf222e] hover:bg-[#ffebe9]"
          >
            Change visibility
          </button>
        </div>

        <div className="mt-4 flex items-start justify-between gap-4 border-t border-[#eaeef2] pt-4">
          <div>
            <div className="text-sm font-semibold">Delete repository</div>
            <p className="text-xs text-[#656d76]">
              Once deleted, there is no going back. Please be certain.
            </p>
          </div>
          <button
            type="button"
            onClick={() => setShowDelete(true)}
            className="rounded-md border border-[#cf222e] bg-white px-3 py-1.5 text-sm font-semibold text-[#cf222e] hover:bg-[#ffebe9]"
          >
            Delete this repository
          </button>
        </div>
      </div>

      {showVisibility && (
        <VisibilityDialog
          owner={owner}
          repo={repo}
          current={meta.visibility ?? (meta.private ? "private" : "public")}
          onClose={() => setShowVisibility(false)}
          onChanged={() => {
            setShowVisibility(false);
            onVisibilityChanged();
          }}
        />
      )}
      {showDelete && (
        <DeleteRepoDialog
          owner={owner}
          repo={repo}
          fullName={`${owner}/${repo}`}
          onClose={() => setShowDelete(false)}
          onDeleted={onDeleted}
        />
      )}
    </div>
  );
}

function VisibilityDialog({
  owner,
  repo,
  current,
  onClose,
  onChanged,
}: {
  owner: string;
  repo: string;
  current: string;
  onClose: () => void;
  onChanged: () => void;
}) {
  const [step, setStep] = useState<1 | 2>(1);
  const [target, setTarget] = useState<"public" | "private" | "internal">(
    current === "private" ? "public" : "private",
  );
  const [confirmText, setConfirmText] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleConfirm = async () => {
    setError(null);
    setSubmitting(true);
    try {
      await jsonFetch(`/repos/${owner}/${repo}`, {
        method: "PATCH",
        body: JSON.stringify({
          visibility: target,
          private: target === "private",
        }),
      });
      onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal onClose={onClose} title="Change repository visibility">
      {step === 1 ? (
        <>
          <p className="text-sm text-[#656d76]">
            Current visibility: <strong>{current}</strong>
          </p>
          <div className="mt-3 space-y-2">
            {(["public", "private", "internal"] as const).map((v) => (
              <label
                key={v}
                className="flex cursor-pointer items-start gap-2 rounded-md border border-[#d0d7de] p-3 hover:bg-[#f6f8fa]"
              >
                <input
                  type="radio"
                  name="visibility-target"
                  checked={target === v}
                  onChange={() => setTarget(v)}
                  className="mt-1"
                />
                <span className="text-sm capitalize">{v}</span>
              </label>
            ))}
          </div>
          <div className="mt-4 flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="rounded-md border border-[#d0d7de] bg-white px-3 py-1.5 text-sm hover:bg-[#f6f8fa]"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={() => setStep(2)}
              disabled={target === current}
              className="rounded-md bg-[#cf222e] px-3 py-1.5 text-sm font-semibold text-white hover:bg-[#a40e26] disabled:opacity-50"
            >
              Continue
            </button>
          </div>
        </>
      ) : (
        <>
          <p className="text-sm">
            Confirm changing visibility from <strong>{current}</strong> to{" "}
            <strong>{target}</strong>.
          </p>
          <p className="mt-2 text-xs text-[#656d76]">
            Type <code className="font-mono">{owner}/{repo}</code> to confirm.
          </p>
          <input
            type="text"
            value={confirmText}
            onChange={(e) => setConfirmText(e.target.value)}
            className="mt-2 w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm font-mono"
          />
          {error && <p className="mt-2 text-sm text-[#cf222e]">{error}</p>}
          <div className="mt-4 flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setStep(1)}
              className="rounded-md border border-[#d0d7de] bg-white px-3 py-1.5 text-sm hover:bg-[#f6f8fa]"
            >
              Back
            </button>
            <button
              type="button"
              onClick={handleConfirm}
              disabled={
                submitting || confirmText.trim() !== `${owner}/${repo}`
              }
              className="rounded-md bg-[#cf222e] px-3 py-1.5 text-sm font-semibold text-white hover:bg-[#a40e26] disabled:opacity-50"
            >
              {submitting ? "Changing…" : "Confirm change"}
            </button>
          </div>
        </>
      )}
    </Modal>
  );
}

function DeleteRepoDialog({
  owner,
  repo,
  fullName,
  onClose,
  onDeleted,
}: {
  owner: string;
  repo: string;
  fullName: string;
  onClose: () => void;
  onDeleted: () => void;
}) {
  const [step, setStep] = useState<1 | 2>(1);
  const [confirmText, setConfirmText] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleDelete = async () => {
    setError(null);
    setSubmitting(true);
    try {
      await jsonFetch(`/repos/${owner}/${repo}`, { method: "DELETE" });
      onDeleted();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal onClose={onClose} title="Delete repository">
      {step === 1 ? (
        <>
          <p className="text-sm">
            This will permanently delete{" "}
            <strong>{fullName}</strong> and all of its contents.
          </p>
          <div className="mt-4 flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="rounded-md border border-[#d0d7de] bg-white px-3 py-1.5 text-sm hover:bg-[#f6f8fa]"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={() => setStep(2)}
              className="rounded-md bg-[#cf222e] px-3 py-1.5 text-sm font-semibold text-white hover:bg-[#a40e26]"
            >
              I understand, continue
            </button>
          </div>
        </>
      ) : (
        <>
          <p className="text-sm">
            Type <code className="font-mono">{fullName}</code> to confirm
            deletion.
          </p>
          <input
            type="text"
            value={confirmText}
            onChange={(e) => setConfirmText(e.target.value)}
            className="mt-2 w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm font-mono"
          />
          {error && <p className="mt-2 text-sm text-[#cf222e]">{error}</p>}
          <div className="mt-4 flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setStep(1)}
              className="rounded-md border border-[#d0d7de] bg-white px-3 py-1.5 text-sm hover:bg-[#f6f8fa]"
            >
              Back
            </button>
            <button
              type="button"
              onClick={handleDelete}
              disabled={submitting || confirmText.trim() !== fullName}
              className="rounded-md bg-[#cf222e] px-3 py-1.5 text-sm font-semibold text-white hover:bg-[#a40e26] disabled:opacity-50"
            >
              {submitting ? "Deleting…" : "Delete this repository"}
            </button>
          </div>
        </>
      )}
    </Modal>
  );
}

function Modal({
  title,
  onClose,
  children,
}: {
  title: string;
  onClose: () => void;
  children: React.ReactNode;
}) {
  return (
    <div className="fixed inset-0 z-[200] flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-md rounded-md border border-[#d0d7de] bg-white shadow-lg">
        <div className="flex items-center justify-between border-b border-[#d0d7de] px-5 py-3">
          <h3 className="text-base font-semibold">{title}</h3>
          <button
            type="button"
            onClick={onClose}
            className="rounded-md px-2 text-[#656d76] hover:bg-[#f6f8fa]"
            aria-label="Close"
          >
            ✕
          </button>
        </div>
        <div className="px-5 py-4">{children}</div>
      </div>
    </div>
  );
}
