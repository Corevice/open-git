"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ApiClient } from "@/lib/api";
import type { OAuthAppWithSecret } from "@/lib/api-types";
import { useAuth } from "@/lib/auth";
import { env } from "@/lib/env";

type CreatedApp = Pick<OAuthAppWithSecret, "id" | "client_id" | "client_secret">;

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

export default function NewOAuthAppPage() {
  const { token } = useAuth();
  const router = useRouter();
  const apiClient = useMemo(
    () => new ApiClient(env.NEXT_PUBLIC_API_BASE_URL, router),
    [router],
  );

  const [name, setName] = useState("");
  const [homepageUrl, setHomepageUrl] = useState("");
  const [callbackUrls, setCallbackUrls] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [createdApp, setCreatedApp] = useState<CreatedApp | null>(null);
  const [copiedField, setCopiedField] = useState<"client_id" | "client_secret" | null>(
    null,
  );

  useEffect(() => {
    if (token) {
      apiClient.setToken(token);
    }
  }, [apiClient, token]);

  useEffect(() => {
    if (!copiedField) {
      return;
    }

    const timer = window.setTimeout(() => setCopiedField(null), 2000);
    return () => window.clearTimeout(timer);
  }, [copiedField]);

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!name.trim() || !homepageUrl.trim() || submitting) {
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

    setSubmitting(true);
    setError(null);
    setCreatedApp(null);
    setCopiedField(null);

    try {
      const result = await apiClient.oauthApps.create({
        name: name.trim(),
        homepage_url: homepageUrl.trim(),
        callback_urls: lines,
        owner_type: "user",
      });

      setCreatedApp({
        id: result.id,
        client_id: result.client_id,
        client_secret: result.client_secret,
      });
      setName("");
      setHomepageUrl("");
      setCallbackUrls("");
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to create OAuth app.",
      );
    } finally {
      setSubmitting(false);
    }
  };

  const handleCopy = async (field: "client_id" | "client_secret", value: string) => {
    try {
      await navigator.clipboard.writeText(value);
      setCopiedField(field);
    } catch {
      setCopiedField(null);
    }
  };

  return (
    <main className="mx-auto max-w-[960px] px-6 py-8">
      <div className="mb-6 border-b border-[#d1d9e0] pb-4">
        <h1 className="mb-2 text-2xl font-semibold">Register a new OAuth App</h1>
        <p className="text-sm text-[#59636e]">
          Create an OAuth application for third-party integrations.
        </p>
      </div>

      {createdApp && (
        <div
          className="mb-6 rounded-md border border-[#54aeff] bg-[#ddf4ff] p-4"
          role="alert"
        >
          <p className="mb-3 text-sm font-semibold text-[#0969da]">
            Copy your client ID and client secret now. You will not be able to see
            the secret again.
          </p>
          <div className="mb-3">
            <span className="text-xs font-medium text-[#59636e]">Client ID</span>
            <code className="mt-1 block break-all rounded bg-white px-3 py-2 text-sm font-mono">
              {createdApp.client_id}
            </code>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => handleCopy("client_id", createdApp.client_id)}
              className="mt-2"
            >
              {copiedField === "client_id" ? "Copied" : "Copy Client ID"}
            </Button>
          </div>
          <div className="mb-3">
            <span className="text-xs font-medium text-[#59636e]">Client Secret</span>
            <p className="mt-1 text-sm text-[#59636e]">
              Your client secret was generated. Use the button below to copy it — it
              is not shown on screen.
            </p>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => handleCopy("client_secret", createdApp.client_secret)}
              className="mt-2"
            >
              {copiedField === "client_secret" ? "Copied" : "Copy Client Secret"}
            </Button>
          </div>
          <Link
            href={`/settings/developers/oauth-apps/${createdApp.id}`}
            className="text-sm text-[#0969da] hover:underline"
          >
            View app settings
          </Link>
        </div>
      )}

      <form
        onSubmit={handleSubmit}
        className="rounded-lg border border-[#d1d9e0] bg-white p-5"
      >
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
        </div>

        <div className="mb-4">
          <Label htmlFor="oauth-app-homepage">Homepage URL</Label>
          <Input
            id="oauth-app-homepage"
            type="url"
            value={homepageUrl}
            onChange={(e) => setHomepageUrl(e.target.value)}
            placeholder="https://example.com"
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
            placeholder="https://example.com/callback&#10;http://localhost:3000/callback"
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

        <div className="flex items-center justify-between">
          <Link
            href="/settings/developers/oauth-apps"
            className="text-sm text-[#0969da] hover:underline"
          >
            Cancel
          </Link>
          <Button
            type="submit"
            disabled={submitting || !name.trim() || !homepageUrl.trim()}
          >
            {submitting ? "Creating…" : "Register application"}
          </Button>
        </div>
      </form>
    </main>
  );
}
