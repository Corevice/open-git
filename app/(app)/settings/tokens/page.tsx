"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  createToken,
  listTokens,
  revokeToken,
  type AccessTokenListItem,
} from "@/lib/api";
import { useAuth } from "@/lib/auth";

const AVAILABLE_SCOPES = ["repo", "read:org", "workflow"] as const;

function formatDate(value: string | null): string {
  if (!value) {
    return "—";
  }
  return new Date(value).toLocaleString();
}

export default function TokensSettingsPage() {
  const { isAuthenticated } = useAuth();
  const router = useRouter();

  const [tokens, setTokens] = useState<AccessTokenListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [revokingId, setRevokingId] = useState<number | null>(null);

  const [note, setNote] = useState("");
  const [selectedScopes, setSelectedScopes] = useState<string[]>([]);
  const [expiresAt, setExpiresAt] = useState("");

  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push("/login");
    }
  }, [isAuthenticated, router]);

  useEffect(() => {
    if (!isAuthenticated) {
      return;
    }

    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(null);
      try {
        const list = await listTokens();
        if (!cancelled) {
          setTokens(list);
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error
              ? err.message
              : "Failed to load personal access tokens.",
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
  }, [isAuthenticated]);

  const toggleScope = (scope: string) => {
    setSelectedScopes((prev) =>
      prev.includes(scope) ? prev.filter((s) => s !== scope) : [...prev, scope],
    );
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
      const result = await createToken({
        note: note.trim(),
        scopes: selectedScopes,
        expires_at: expiresAt
          ? new Date(`${expiresAt}T23:59:59`).toISOString()
          : undefined,
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
      setNote("");
      setSelectedScopes([]);
      setExpiresAt("");
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to create personal access token.",
      );
    } finally {
      setSubmitting(false);
    }
  };

  const handleRevoke = async (id: number) => {
    setRevokingId(id);
    setError(null);

    try {
      await revokeToken(id);
      setTokens((prev) => prev.filter((t) => t.id !== id));
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to revoke personal access token.",
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

  if (!isAuthenticated) {
    return null;
  }

  return (
    <main className="mx-auto max-w-[960px] px-6 py-8">
      <div className="mb-6 border-b border-[#d1d9e0] pb-4">
        <h1 className="mb-2 text-2xl font-semibold">Personal Access Tokens</h1>
        <p className="text-sm text-[#59636e]">
          Create tokens for gh auth login. The raw token value is shown only once
          after creation.
        </p>
      </div>

      <nav className="mb-6 flex gap-4 text-sm">
        <Link href="/settings/profile" className="text-[#0969da] hover:underline">
          Profile
        </Link>
        <span className="font-semibold text-[#0969da]">Personal Access Tokens</span>
      </nav>

      {createdToken && (
        <div
          className="mb-6 rounded-md border border-[#54aeff] bg-[#ddf4ff] p-4"
          role="alert"
        >
          <p className="mb-2 text-sm font-semibold text-[#0969da]">
            Copy your token now. You will not be able to see it again.
          </p>
          <code className="block break-all rounded bg-white px-3 py-2 text-sm font-mono">
            {createdToken}
          </code>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleCopy}
            className="mt-3"
          >
            {copied ? "Copied" : "Copy"}
          </Button>
        </div>
      )}

      <form
        onSubmit={handleCreate}
        className="mb-6 rounded-lg border border-[#d1d9e0] bg-white p-5"
      >
        <h2 className="mb-4 text-lg font-semibold">Create token</h2>

        <div className="mb-4">
          <Label htmlFor="token-note">Note</Label>
          <Input
            id="token-note"
            type="text"
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="gh-cli-token"
            required
            className="mt-1.5"
          />
        </div>

        <div className="mb-4">
          <span className="text-sm font-medium">Scopes</span>
          <div className="mt-2 grid gap-2 sm:grid-cols-3">
            {AVAILABLE_SCOPES.map((scope) => (
              <label
                key={scope}
                className="flex cursor-pointer items-center gap-2 text-sm"
              >
                <input
                  type="checkbox"
                  checked={selectedScopes.includes(scope)}
                  onChange={() => toggleScope(scope)}
                />
                {scope}
              </label>
            ))}
          </div>
        </div>

        <div className="mb-4">
          <Label htmlFor="token-expires">Expiry (optional)</Label>
          <Input
            id="token-expires"
            type="date"
            value={expiresAt}
            onChange={(e) => setExpiresAt(e.target.value)}
            className="mt-1.5 max-w-xs"
          />
        </div>

        <div className="flex justify-end">
          <Button
            type="submit"
            disabled={submitting || !note.trim() || selectedScopes.length === 0}
          >
            {submitting ? "Creating…" : "Create token"}
          </Button>
        </div>
      </form>

      {error && (
        <p className="mb-4 text-sm text-[#cf222e]" role="alert">
          {error}
        </p>
      )}

      {loading ? (
        <p className="text-sm text-[#59636e]">Loading…</p>
      ) : tokens.length === 0 ? (
        <p className="text-sm text-[#59636e]">No personal access tokens yet.</p>
      ) : (
        <div className="overflow-x-auto rounded-lg border border-[#d1d9e0]">
          <table className="min-w-full text-left text-sm">
            <thead className="border-b border-[#d1d9e0] bg-[#f6f8fa]">
              <tr>
                <th className="px-4 py-3 font-semibold">Note</th>
                <th className="px-4 py-3 font-semibold">Scopes</th>
                <th className="px-4 py-3 font-semibold">Created At</th>
                <th className="px-4 py-3 font-semibold">Last Used</th>
                <th className="px-4 py-3 font-semibold">Expires</th>
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
                  <td className="px-4 py-3">{formatDate(item.created_at)}</td>
                  <td className="px-4 py-3">{formatDate(item.last_used_at)}</td>
                  <td className="px-4 py-3">{formatDate(item.expires_at)}</td>
                  <td className="px-4 py-3">
                    <Button
                      type="button"
                      variant="destructive"
                      size="sm"
                      onClick={() => handleRevoke(item.id)}
                      disabled={revokingId === item.id || item.revoked_at !== null}
                    >
                      {revokingId === item.id ? "Revoking…" : "Revoke"}
                    </Button>
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
