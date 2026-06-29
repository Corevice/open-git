"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { z } from "zod";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { API_TOKEN_KEY, ApiClient, ApiError } from "@/lib/api";
import type { OAuthApp } from "@/lib/api-types";
import { useAuth } from "@/lib/auth";

const oauthAppFormSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, "Name is required")
    .max(100, "Name must be at most 100 characters"),
  url: z.union([
    z.literal(""),
    z.string().url("Enter a valid homepage URL"),
  ]),
  callback_url: z.string().url("Enter a valid callback URL"),
});

type FormErrors = Partial<Record<"name" | "url" | "callback_url", string>>;

function formatDate(value: string): string {
  return new Date(value).toLocaleString();
}

function createApiClient(): ApiClient {
  const baseURL =
    process.env.NEXT_PUBLIC_API_BASE_URL ??
    process.env.NEXT_PUBLIC_API_URL ??
    "http://localhost:8080";
  const client = new ApiClient(baseURL);
  if (typeof window !== "undefined") {
    const storedToken = localStorage.getItem(API_TOKEN_KEY);
    if (storedToken) {
      client.setToken(storedToken);
    }
  }
  return client;
}

export default function OAuthAppsSettingsPage() {
  const { isAuthenticated } = useAuth();
  const router = useRouter();
  const api = useMemo(() => createApiClient(), []);

  const [apps, setApps] = useState<OAuthApp[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [newAppSecret, setNewAppSecret] = useState<string | null>(null);

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [regeneratingId, setRegeneratingId] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [formErrors, setFormErrors] = useState<FormErrors>({});

  const [name, setName] = useState("");
  const [url, setUrl] = useState("");
  const [callbackUrl, setCallbackUrl] = useState("");

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
        const list = await api.oauthApps.list();
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
  }, [api, isAuthenticated]);

  const handleCreate = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (submitting) {
      return;
    }

    const parsed = oauthAppFormSchema.safeParse({
      name,
      url,
      callback_url: callbackUrl,
    });

    if (!parsed.success) {
      const nextErrors: FormErrors = {};
      for (const issue of parsed.error.issues) {
        const field = issue.path[0];
        if (
          field === "name" ||
          field === "url" ||
          field === "callback_url"
        ) {
          nextErrors[field] = issue.message;
        }
      }
      setFormErrors(nextErrors);
      return;
    }

    setFormErrors({});
    setSubmitting(true);
    setError(null);
    setCopied(false);

    try {
      const result = await api.oauthApps.create({
        name: parsed.data.name,
        homepage_url: parsed.data.url,
        callback_urls: [parsed.data.callback_url],
        owner_type: "user",
      });

      setNewAppSecret(result.client_secret);
      setApps((prev) => [result, ...prev]);
      setName("");
      setUrl("");
      setCallbackUrl("");
      setShowForm(false);
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : "Failed to register OAuth app.",
      );
    } finally {
      setSubmitting(false);
    }
  };

  const handleRegenerateSecret = async (clientId: string) => {
    setRegeneratingId(clientId);
    setError(null);
    setCopied(false);

    try {
      const result = await api.oauthApps.regenerateSecret(clientId);
      setNewAppSecret(result.client_secret);
    } catch (err) {
      setError(
        err instanceof ApiError
          ? err.message
          : "Failed to regenerate client secret.",
      );
    } finally {
      setRegeneratingId(null);
    }
  };

  const handleDelete = async (clientId: string, appName: string) => {
    const confirmed = window.confirm(
      `Delete OAuth app "${appName}"? This action cannot be undone.`,
    );
    if (!confirmed) {
      return;
    }

    setDeletingId(clientId);
    setError(null);

    try {
      await api.oauthApps.delete(clientId);
      setApps((prev) => prev.filter((app) => app.client_id !== clientId));
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : "Failed to delete OAuth app.",
      );
    } finally {
      setDeletingId(null);
    }
  };

  const handleCopySecret = async () => {
    if (!newAppSecret) {
      return;
    }
    try {
      await navigator.clipboard.writeText(newAppSecret);
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
        <h1 className="mb-2 text-2xl font-semibold">OAuth Apps</h1>
        <p className="text-sm text-[#59636e]">
          Register OAuth applications and manage client credentials for
          authorization code flows.
        </p>
      </div>

      <nav className="mb-6 flex gap-4 text-sm">
        <Link href="/settings/profile" className="text-[#0969da] hover:underline">
          Profile
        </Link>
        <Link href="/settings/tokens" className="text-[#0969da] hover:underline">
          Personal Access Tokens
        </Link>
        <span className="font-semibold text-[#0969da]">OAuth Apps</span>
      </nav>

      {newAppSecret !== null && (
        <Alert className="mb-6 border-amber-300 bg-amber-50 text-amber-950">
          <AlertTitle>Your client secret will only be shown once. Copy it now.</AlertTitle>
          <AlertDescription>
            <code className="mt-2 block break-all rounded bg-white px-3 py-2 font-mono text-sm text-[#24292f]">
              {newAppSecret}
            </code>
            <div className="mt-3 flex gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleCopySecret}
              >
                {copied ? "Copied" : "Copy"}
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => {
                  setNewAppSecret(null);
                  setCopied(false);
                }}
              >
                Dismiss
              </Button>
            </div>
          </AlertDescription>
        </Alert>
      )}

      {!showForm ? (
        <div className="mb-6">
          <Button type="button" onClick={() => setShowForm(true)}>
            Register new OAuth app
          </Button>
        </div>
      ) : (
        <form
          onSubmit={handleCreate}
          className="mb-6 rounded-lg border border-[#d1d9e0] bg-white p-5"
        >
          <h2 className="mb-4 text-lg font-semibold">Register OAuth app</h2>

          <div className="mb-4">
            <Label htmlFor="oauth-app-name">Application name</Label>
            <Input
              id="oauth-app-name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My OAuth App"
              required
              className="mt-1.5"
            />
            {formErrors.name && (
              <p className="mt-1 text-sm text-[#cf222e]">{formErrors.name}</p>
            )}
          </div>

          <div className="mb-4">
            <Label htmlFor="oauth-app-url">Homepage URL</Label>
            <Input
              id="oauth-app-url"
              type="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://example.com"
              className="mt-1.5"
            />
            {formErrors.url && (
              <p className="mt-1 text-sm text-[#cf222e]">{formErrors.url}</p>
            )}
          </div>

          <div className="mb-4">
            <Label htmlFor="oauth-app-callback">Authorization callback URL</Label>
            <Input
              id="oauth-app-callback"
              type="url"
              value={callbackUrl}
              onChange={(e) => setCallbackUrl(e.target.value)}
              placeholder="https://example.com/oauth/callback"
              required
              className="mt-1.5"
            />
            {formErrors.callback_url && (
              <p className="mt-1 text-sm text-[#cf222e]">
                {formErrors.callback_url}
              </p>
            )}
          </div>

          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setShowForm(false);
                setFormErrors({});
              }}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? "Registering…" : "Register application"}
            </Button>
          </div>
        </form>
      )}

      {error && (
        <p className="mb-4 text-sm text-[#cf222e]" role="alert">
          {error}
        </p>
      )}

      {loading ? (
        <p className="text-sm text-[#59636e]">Loading…</p>
      ) : apps.length === 0 ? (
        <p className="text-sm text-[#59636e]">No OAuth apps registered yet.</p>
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
                <tr
                  key={app.client_id}
                  className="border-b border-[#d1d9e0] last:border-b-0"
                >
                  <td className="px-4 py-3 font-medium">{app.name}</td>
                  <td className="px-4 py-3 font-mono text-xs">{app.client_id}</td>
                  <td className="px-4 py-3">{formatDate(app.created_at)}</td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-2">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => handleRegenerateSecret(app.client_id)}
                        disabled={regeneratingId === app.client_id}
                      >
                        {regeneratingId === app.client_id
                          ? "Regenerating…"
                          : "Regenerate Secret"}
                      </Button>
                      <Button
                        type="button"
                        variant="destructive"
                        size="sm"
                        onClick={() => handleDelete(app.client_id, app.name)}
                        disabled={deletingId === app.client_id}
                      >
                        {deletingId === app.client_id ? "Deleting…" : "Delete"}
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
