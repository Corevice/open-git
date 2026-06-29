"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { FormEvent, useEffect, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ApiClient } from "@/lib/api";
import type { OAuthApp } from "@/lib/api-types";
import { useAuth } from "@/lib/auth";
import { env } from "@/lib/env";

function validateCallbackUrls(urls: string[]): string | null {
  for (const url of urls) {
    try {
      const parsed = new URL(url);
      if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
        return `Invalid callback URL "${url}": only http and https URLs are allowed.`;
      }
    } catch {
      return `Invalid callback URL "${url}".`;
    }
  }
  return null;
}

export default function OAuthAppDetailPage() {
  const { token } = useAuth();
  const router = useRouter();
  const params = useParams();
  const id = typeof params.id === "string" ? params.id : "";

  const apiClient = useMemo(
    () => new ApiClient(env.NEXT_PUBLIC_API_BASE_URL, router),
    [router],
  );

  const [app, setApp] = useState<OAuthApp | null>(null);
  const [name, setName] = useState("");
  const [homepageUrl, setHomepageUrl] = useState("");
  const [callbackUrls, setCallbackUrls] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [regenerating, setRegenerating] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [regeneratedSecret, setRegeneratedSecret] = useState<string | null>(null);
  const [copiedSecret, setCopiedSecret] = useState(false);

  useEffect(() => {
    if (token) {
      apiClient.setToken(token);
    }
  }, [apiClient, token]);

  useEffect(() => {
    if (!token || !id) {
      setLoading(false);
      return;
    }

    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(null);
      try {
        const oauthApp = await apiClient.oauthApps.get(id);
        if (!cancelled) {
          setApp(oauthApp);
          setName(oauthApp.name);
          setHomepageUrl(oauthApp.homepage_url);
          setCallbackUrls(oauthApp.callback_urls.join("\n"));
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error ? err.message : "Failed to load OAuth app.",
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
  }, [apiClient, token, id]);

  useEffect(() => {
    if (!copiedSecret) {
      return;
    }

    const timer = window.setTimeout(() => setCopiedSecret(false), 2000);
    return () => window.clearTimeout(timer);
  }, [copiedSecret]);

  const handleSave = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!id || saving) {
      return;
    }

    const lines = callbackUrls
      .split("\n")
      .map((line) => line.trim())
      .filter(Boolean);

    const callbackError = validateCallbackUrls(lines);
    if (callbackError) {
      setError(callbackError);
      return;
    }

    setSaving(true);
    setError(null);
    setSuccessMessage(null);

    try {
      const updated = await apiClient.oauthApps.update(id, {
        name: name.trim(),
        homepage_url: homepageUrl.trim(),
        callback_urls: lines,
      });
      setApp(updated);
      setName(updated.name);
      setHomepageUrl(updated.homepage_url);
      setCallbackUrls(updated.callback_urls.join("\n"));
      setSuccessMessage("Changes saved.");
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to update OAuth app.",
      );
    } finally {
      setSaving(false);
    }
  };

  const handleRegenerateSecret = async () => {
    if (!id || regenerating || !app) {
      return;
    }

    if (
      !window.confirm(
        `Regenerate the client secret for "${app.name}"? The current secret will stop working immediately.`,
      )
    ) {
      return;
    }

    setRegenerating(true);
    setError(null);
    setRegeneratedSecret(null);
    setCopiedSecret(false);

    try {
      const result = await apiClient.oauthApps.regenerateSecret(id);
      setRegeneratedSecret(result.client_secret);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to regenerate client secret.",
      );
    } finally {
      setRegenerating(false);
    }
  };

  const handleDelete = async () => {
    if (!id || deleting || !app) {
      return;
    }

    if (
      !window.confirm(
        `Delete OAuth app "${app.name}"? This will permanently revoke all associated access tokens and cannot be undone.`,
      )
    ) {
      return;
    }

    setDeleting(true);
    setError(null);

    try {
      await apiClient.oauthApps.delete(id);
      router.push("/settings/developers/oauth-apps");
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to delete OAuth app.",
      );
      setDeleting(false);
    }
  };

  const handleCopySecret = async () => {
    if (!regeneratedSecret) {
      return;
    }
    try {
      await navigator.clipboard.writeText(regeneratedSecret);
      setCopiedSecret(true);
    } catch {
      setCopiedSecret(false);
    }
  };

  const dismissSecretAlert = () => {
    setRegeneratedSecret(null);
    setCopiedSecret(false);
  };

  if (loading) {
    return (
      <main className="mx-auto max-w-[960px] px-6 py-8">
        <p className="text-sm text-[#59636e]">Loading…</p>
      </main>
    );
  }

  if (!app) {
    return (
      <main className="mx-auto max-w-[960px] px-6 py-8">
        <p className="text-sm text-[#cf222e]" role="alert">
          {error ?? "OAuth app not found."}
        </p>
        <Link
          href="/settings/developers/oauth-apps"
          className="mt-4 inline-block text-sm text-[#0969da] hover:underline"
        >
          Back to OAuth Apps
        </Link>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-[960px] px-6 py-8">
      <div className="mb-6 border-b border-[#d1d9e0] pb-4">
        <h1 className="mb-2 text-2xl font-semibold">{app.name}</h1>
        <p className="text-sm text-[#59636e]">Manage your OAuth application settings.</p>
      </div>

      <Link
        href="/settings/developers/oauth-apps"
        className="mb-6 inline-block text-sm text-[#0969da] hover:underline"
      >
        ← Back to OAuth Apps
      </Link>

      {regeneratedSecret && (
        <div
          className="mb-6 rounded-md border border-[#54aeff] bg-[#ddf4ff] p-4"
          role="alert"
        >
          <p className="mb-2 text-sm font-semibold text-[#0969da]">
            Copy your new client secret now. You will not be able to see it again.
          </p>
          <p className="mb-3 text-sm text-[#59636e]">
            Your new client secret was generated. Use the button below to copy it — it
            is not shown on screen.
          </p>
          <div className="flex gap-2">
            <Button type="button" variant="outline" size="sm" onClick={handleCopySecret}>
              {copiedSecret ? "Copied" : "Copy"}
            </Button>
            <Button type="button" variant="outline" size="sm" onClick={dismissSecretAlert}>
              Dismiss
            </Button>
          </div>
        </div>
      )}

      <form
        onSubmit={handleSave}
        className="mb-8 rounded-lg border border-[#d1d9e0] bg-white p-5"
      >
        <div className="mb-4">
          <Label htmlFor="oauth-client-id">Client ID</Label>
          <code
            id="oauth-client-id"
            className="mt-1.5 block break-all rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-3 py-2 text-sm font-mono"
          >
            {app.client_id}
          </code>
        </div>

        <div className="mb-4">
          <Label htmlFor="oauth-app-name">Application name</Label>
          <Input
            id="oauth-app-name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            className="mt-1.5"
          />
        </div>

        <div className="mb-4">
          <Label htmlFor="oauth-app-homepage">Homepage URL</Label>
          <Input
            id="oauth-app-homepage"
            type="url"
            value={homepageUrl}
            onChange={(e) => setHomepageUrl(e.target.value)}
            required
            className="mt-1.5"
          />
        </div>

        <div className="mb-4">
          <Label htmlFor="oauth-app-callbacks">Authorization callback URLs</Label>
          <textarea
            id="oauth-app-callbacks"
            value={callbackUrls}
            onChange={(e) => setCallbackUrls(e.target.value)}
            rows={4}
            className="mt-1.5 w-full rounded-md border border-[#d1d9e0] px-3 py-2 text-sm focus:border-[#0969da] focus:outline-none focus:ring-1 focus:ring-[#0969da]"
          />
          <p className="mt-1 text-xs text-[#59636e]">Enter one URL per line.</p>
        </div>

        {error && (
          <p className="mb-4 text-sm text-[#cf222e]" role="alert">
            {error}
          </p>
        )}

        {successMessage && (
          <p className="mb-4 text-sm text-[#1a7f37]" role="status">
            {successMessage}
          </p>
        )}

        <div className="flex justify-end">
          <Button type="submit" disabled={saving || !name.trim() || !homepageUrl.trim()}>
            {saving ? "Saving…" : "Save changes"}
          </Button>
        </div>
      </form>

      <section className="rounded-lg border border-[#cf222e] bg-white p-5">
        <h2 className="mb-2 text-lg font-semibold text-[#cf222e]">Danger zone</h2>
        <p className="mb-4 text-sm text-[#59636e]">
          Regenerating the client secret will invalidate the previous secret. Deleting
          the app cannot be undone.
        </p>

        <div className="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-md border border-[#d1d9e0] p-4">
          <div>
            <p className="text-sm font-medium">Regenerate client secret</p>
            <p className="text-xs text-[#59636e]">
              Generate a new client secret. The old secret will stop working immediately.
            </p>
          </div>
          <Button
            type="button"
            variant="destructive"
            size="sm"
            onClick={handleRegenerateSecret}
            disabled={regenerating}
          >
            {regenerating ? "Regenerating…" : "Regenerate client secret"}
          </Button>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-[#d1d9e0] p-4">
          <div>
            <p className="text-sm font-medium">Delete this OAuth App</p>
            <p className="text-xs text-[#59636e]">
              Permanently remove this application and revoke all associated tokens.
            </p>
          </div>
          <Button
            type="button"
            variant="destructive"
            size="sm"
            onClick={handleDelete}
            disabled={deleting}
          >
            {deleting ? "Deleting…" : "Delete this OAuth App"}
          </Button>
        </div>
      </section>
    </main>
  );
}
