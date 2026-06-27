"use client";

import Link from "next/link";
import { FormEvent, useEffect, useMemo, useState } from "react";

import { ApiClient, type AccessTokenListItem } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const AVAILABLE_SCOPES = [
  { value: "repo", label: "repo" },
  { value: "read:org", label: "read:org" },
  { value: "admin:org", label: "admin:org" },
  { value: "user", label: "user" },
] as const;

type ExpiryOption = "none" | "7d" | "30d" | "90d";

function computeExpiresAt(option: ExpiryOption): string | undefined {
  if (option === "none") {
    return undefined;
  }

  const days = option === "7d" ? 7 : option === "30d" ? 30 : 90;
  const date = new Date();
  date.setDate(date.getDate() + days);
  return date.toISOString();
}

function formatDate(value: string | null): string {
  if (!value) {
    return "—";
  }
  return new Date(value).toLocaleString();
}

export default function TokensSettingsPage() {
  const { token } = useAuth();
  const baseURL =
    process.env.NEXT_PUBLIC_API_BASE_URL ??
    process.env.NEXT_PUBLIC_API_URL ??
    "http://localhost:8080";

  const apiClient = useMemo(() => new ApiClient(baseURL), [baseURL]);

  const [tokens, setTokens] = useState<AccessTokenListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [revokingId, setRevokingId] = useState<number | null>(null);

  const [note, setNote] = useState("");
  const [selectedScopes, setSelectedScopes] = useState<string[]>([]);
  const [expiry, setExpiry] = useState<ExpiryOption>("30d");

  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (token) {
      apiClient.setToken(token);
    }
  }, [apiClient, token]);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(null);
      try {
        const list = await apiClient.tokens.list();
        if (!cancelled) {
          setTokens(list);
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error
              ? err.message
              : "トークン一覧の読み込みに失敗しました。",
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
  }, [apiClient]);

  const toggleScope = (scope: string) => {
    setSelectedScopes((prev) =>
      prev.includes(scope) ? prev.filter((s) => s !== scope) : [...prev, scope],
    );
  };

  const resetForm = () => {
    setNote("");
    setSelectedScopes([]);
    setExpiry("30d");
    setShowForm(false);
  };

  const handleCreate = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!note.trim() || selectedScopes.length === 0 || submitting) {
      return;
    }

    setSubmitting(true);
    setError(null);
    setCreatedToken(null);
    setCopied(false);

    try {
      const result = await apiClient.tokens.create({
        note: note.trim(),
        scopes: selectedScopes,
        expires_at: computeExpiresAt(expiry),
      });

      setCreatedToken(result.token);
      setTokens((prev) => [
        {
          id: result.id,
          note: result.note,
          scopes: result.scopes,
          expires_at: result.expires_at,
          created_at: result.created_at,
          last_used_at: result.last_used_at,
          revoked_at: null,
        },
        ...prev,
      ]);
      resetForm();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "トークンの発行に失敗しました。",
      );
    } finally {
      setSubmitting(false);
    }
  };

  const handleRevoke = async (id: number) => {
    setRevokingId(id);
    setError(null);

    try {
      await apiClient.tokens.revoke(id);
      setTokens((prev) => prev.filter((t) => t.id !== id));
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "トークンの失効に失敗しました。",
      );
    } finally {
      setRevokingId(null);
    }
  };

  const handleCopy = async () => {
    if (!createdToken) {
      return;
    }
    try {
      await navigator.clipboard.writeText(createdToken);
      setCopied(true);
    } catch {
      setCopied(false);
    }
  };

  return (
    <main className="mx-auto max-w-[960px] px-6 py-8">
      <div className="mb-6 flex items-start justify-between border-b border-[#d1d9e0] pb-4">
        <div>
          <h1 className="mb-2 text-2xl font-semibold">
            Personal Access Tokens
          </h1>
          <p className="text-sm text-[#59636e]">
            API アクセス用のトークンを発行・管理します。平文は発行時に一度だけ表示されます。
          </p>
        </div>
        {!showForm && (
          <button
            type="button"
            onClick={() => setShowForm(true)}
            className="rounded-md bg-[#1f883d] px-4 py-2 text-sm font-medium text-white hover:bg-[#1a7f37]"
          >
            New token
          </button>
        )}
      </div>

      <nav className="mb-6 flex gap-4 text-sm">
        <Link href="/settings/profile" className="text-[#0969da] hover:underline">
          プロフィール
        </Link>
        <span className="font-semibold text-[#0969da]">
          Personal Access Tokens
        </span>
      </nav>

      {createdToken && (
        <div
          className="mb-6 rounded-md border border-[#54aeff] bg-[#ddf4ff] p-4"
          role="alert"
        >
          <div className="mb-2 flex items-start justify-between gap-4">
            <p className="text-sm font-semibold text-[#0969da]">
              トークンを発行しました。この平文は再表示できません。
            </p>
            <button
              type="button"
              onClick={() => setCreatedToken(null)}
              className="text-sm text-[#59636e] hover:text-[#24292f]"
              aria-label="Dismiss"
            >
              ✕
            </button>
          </div>
          <code className="block break-all rounded bg-white px-3 py-2 text-sm font-mono">
            {createdToken}
          </code>
          <button
            type="button"
            onClick={handleCopy}
            className="mt-3 rounded-md border border-[#d1d9e0] bg-white px-3 py-1.5 text-sm hover:bg-[#f6f8fa]"
          >
            {copied ? "コピーしました" : "クリップボードにコピー"}
          </button>
        </div>
      )}

      {showForm && (
        <form
          onSubmit={handleCreate}
          className="mb-6 rounded-lg border border-[#d1d9e0] bg-white p-5"
        >
          <h2 className="mb-4 text-lg font-semibold">新しいトークン</h2>

          <div className="mb-4">
            <label htmlFor="token-note" className="mb-1.5 block text-sm font-semibold">
              Note <span className="text-[#cf222e]">*</span>
            </label>
            <input
              id="token-note"
              type="text"
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="例: ci-deploy-token"
              className="w-full rounded-md border border-[#d1d9e0] px-3 py-2 text-sm"
              required
            />
          </div>

          <div className="mb-4">
            <span className="mb-2 block text-sm font-semibold">Scopes</span>
            <div className="grid gap-2 sm:grid-cols-2">
              {AVAILABLE_SCOPES.map((scope) => (
                <label
                  key={scope.value}
                  className="flex cursor-pointer items-center gap-2 text-sm"
                >
                  <input
                    type="checkbox"
                    checked={selectedScopes.includes(scope.value)}
                    onChange={() => toggleScope(scope.value)}
                  />
                  {scope.label}
                </label>
              ))}
            </div>
          </div>

          <div className="mb-4">
            <span className="mb-2 block text-sm font-semibold">有効期限</span>
            <div className="space-y-2">
              {(
                [
                  ["none", "無期限"],
                  ["7d", "7日"],
                  ["30d", "30日"],
                  ["90d", "90日"],
                ] as const
              ).map(([value, label]) => (
                <label
                  key={value}
                  className="flex cursor-pointer items-center gap-2 text-sm"
                >
                  <input
                    type="radio"
                    name="expiry"
                    checked={expiry === value}
                    onChange={() => setExpiry(value)}
                  />
                  {label}
                </label>
              ))}
            </div>
          </div>

          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={resetForm}
              className="rounded-md border border-[#d1d9e0] bg-white px-4 py-2 text-sm hover:bg-[#f6f8fa]"
            >
              キャンセル
            </button>
            <button
              type="submit"
              disabled={submitting || !note.trim() || selectedScopes.length === 0}
              className="rounded-md bg-[#1f883d] px-4 py-2 text-sm font-medium text-white hover:bg-[#1a7f37] disabled:cursor-not-allowed disabled:opacity-50"
            >
              {submitting ? "発行中…" : "トークンを発行"}
            </button>
          </div>
        </form>
      )}

      {error && (
        <p className="mb-4 text-sm text-[#cf222e]" role="alert">
          {error}
        </p>
      )}

      {loading ? (
        <p className="text-sm text-[#59636e]">読み込み中…</p>
      ) : tokens.length === 0 ? (
        <p className="text-sm text-[#59636e]">発行済みトークンはありません。</p>
      ) : (
        <div className="overflow-x-auto rounded-lg border border-[#d1d9e0]">
          <table className="min-w-full text-left text-sm">
            <thead className="border-b border-[#d1d9e0] bg-[#f6f8fa]">
              <tr>
                <th className="px-4 py-3 font-semibold">Note</th>
                <th className="px-4 py-3 font-semibold">Scopes</th>
                <th className="px-4 py-3 font-semibold">Expires</th>
                <th className="px-4 py-3 font-semibold">Revoked</th>
                <th className="px-4 py-3 font-semibold">Actions</th>
              </tr>
            </thead>
            <tbody>
              {tokens.map((item) => (
                <tr key={item.id} className="border-b border-[#d1d9e0] last:border-b-0">
                  <td className="px-4 py-3 font-medium">{item.note}</td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      {item.scopes.map((scope) => (
                        <span
                          key={scope}
                          className="rounded bg-[#f6f8fa] px-2 py-0.5 font-mono text-xs text-[#59636e]"
                        >
                          {scope}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="px-4 py-3">{formatDate(item.expires_at)}</td>
                  <td className="px-4 py-3">{formatDate(item.revoked_at)}</td>
                  <td className="px-4 py-3">
                    <button
                      type="button"
                      onClick={() => handleRevoke(item.id)}
                      disabled={revokingId === item.id || item.revoked_at !== null}
                      className="rounded-md bg-[#cf222e] px-3 py-1 text-xs font-medium text-white hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      {revokingId === item.id ? "失効中…" : "Revoke"}
                    </button>
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
