"use client";

import { FormEvent, use, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  getWebhook,
  isWebhookValidationError,
  updateWebhook,
  type UpdateWebhookPayload,
  type Webhook,
} from "@/lib/api/webhooks";

const INDIVIDUAL_EVENTS = [
  "push",
  "pull_request",
  "issues",
  "issue_comment",
  "release",
  "create",
  "delete",
  "workflow_run",
] as const;

function isValidHttpUrl(value: string): boolean {
  try {
    const parsed = new URL(value);
    return parsed.protocol === "http:" || parsed.protocol === "https:";
  } catch {
    return false;
  }
}

function eventsEqual(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false;
  const sortedA = [...a].sort();
  const sortedB = [...b].sort();
  return sortedA.every((value, index) => value === sortedB[index]);
}

export default function EditWebhookPage({
  params,
}: {
  params: Promise<{ owner: string; repo: string; hookId: string }>;
}) {
  const { owner, repo, hookId: hookIdParam } = use(params);
  const hookId = Number(hookIdParam);
  const router = useRouter();

  const [initial, setInitial] = useState<Webhook | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [url, setUrl] = useState("");
  const [contentType, setContentType] = useState<"json" | "form">("json");
  const [secret, setSecret] = useState("");
  const [sendEverything, setSendEverything] = useState(false);
  const [selectedEvents, setSelectedEvents] = useState<Set<string>>(new Set());
  const [active, setActive] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      setLoadError(null);
      try {
        const webhook = await getWebhook(owner, repo, hookId);
        if (cancelled) return;
        setInitial(webhook);
        setUrl(webhook.config.url);
        setContentType(webhook.config.content_type);
        const allEvents = webhook.events.includes("*");
        setSendEverything(allEvents);
        setSelectedEvents(
          allEvents ? new Set() : new Set(webhook.events),
        );
        setActive(webhook.active);
      } catch (err) {
        if (!cancelled) {
          setLoadError(
            err instanceof Error ? err.message : "Failed to load webhook.",
          );
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => {
      cancelled = true;
    };
  }, [owner, repo, hookId]);

  const toggleEvent = (event: string) => {
    setSelectedEvents((prev) => {
      const next = new Set(prev);
      if (next.has(event)) {
        next.delete(event);
      } else {
        next.add(event);
      }
      return next;
    });
  };

  const handleSendEverythingChange = (checked: boolean) => {
    setSendEverything(checked);
    if (checked) {
      setSelectedEvents(new Set());
    } else if (initial) {
      setSelectedEvents(
        initial.events.includes("*")
          ? new Set(["push"])
          : new Set(initial.events),
      );
    }
  };

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    if (!initial) return;

    setFormError(null);
    setFieldErrors({});

    const trimmedUrl = url.trim();
    if (!trimmedUrl) {
      setFieldErrors({ "config.url": "URL is required." });
      return;
    }
    if (!isValidHttpUrl(trimmedUrl)) {
      setFieldErrors({
        "config.url": "URL must use http:// or https:// scheme.",
      });
      return;
    }

    const events = sendEverything ? ["*"] : Array.from(selectedEvents);
    if (events.length === 0) {
      setFieldErrors({ events: "Select at least one event." });
      return;
    }

    const payload: UpdateWebhookPayload = {};
    const configChanges: NonNullable<UpdateWebhookPayload["config"]> = {};

    if (trimmedUrl !== initial.config.url) {
      configChanges.url = trimmedUrl;
    }
    if (contentType !== initial.config.content_type) {
      configChanges.content_type = contentType;
    }
    if (secret.trim()) {
      configChanges.secret = secret.trim();
    }
    if (Object.keys(configChanges).length > 0) {
      payload.config = configChanges;
    }

    if (active !== initial.active) {
      payload.active = active;
    }

    if (!eventsEqual(events, initial.events)) {
      payload.events = events;
    }

    if (Object.keys(payload).length === 0) {
      router.push(`/${owner}/${repo}/settings/hooks`);
      return;
    }

    setSubmitting(true);
    try {
      await updateWebhook(owner, repo, hookId, payload);
      router.push(`/${owner}/${repo}/settings/hooks`);
    } catch (err) {
      if (isWebhookValidationError(err)) {
        setFormError(err.message);
        setFieldErrors(err.fieldErrors);
      } else if (err instanceof Error) {
        setFormError(err.message);
      } else {
        setFormError("Failed to update webhook.");
      }
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="mx-auto max-w-2xl px-6 py-8">
        <p className="text-sm text-[#656d76]">Loading…</p>
      </div>
    );
  }

  if (loadError || !initial) {
    return (
      <div className="mx-auto max-w-2xl px-6 py-8">
        <p className="text-sm text-[#cf222e]">
          {loadError ?? "Webhook not found."}
        </p>
        <Button variant="outline" asChild className="mt-4">
          <Link href={`/${owner}/${repo}/settings/hooks`}>Back to webhooks</Link>
        </Button>
      </div>
    );
  }

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
          href={`/${owner}/${repo}/settings/hooks`}
          className="rounded-md px-3 py-1.5 text-sm hover:bg-[#f6f8fa]"
        >
          ← Webhooks
        </Link>
      </header>

      <div className="mx-auto max-w-2xl px-6 py-6">
        <div className="mb-4 text-sm text-[#656d76]">
          <Link href={`/${owner}/${repo}`} className="text-[#0969da]">
            {owner}/{repo}
          </Link>{" "}
          /{" "}
          <Link
            href={`/${owner}/${repo}/settings/hooks`}
            className="text-[#0969da]"
          >
            Webhooks
          </Link>{" "}
          / Edit webhook
        </div>

        <h1 className="mb-6 text-2xl font-semibold">Edit webhook</h1>

        <form
          onSubmit={handleSubmit}
          className="space-y-6 rounded-md border border-[#d0d7de] bg-white p-6"
        >
          <div className="space-y-2">
            <Label htmlFor="webhook-url">
              Payload URL <span className="text-[#cf222e]">*</span>
            </Label>
            <Input
              id="webhook-url"
              type="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://example.com/webhook"
              required
            />
            {fieldErrors["config.url"] && (
              <p className="text-sm text-[#cf222e]">
                {fieldErrors["config.url"]}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="content-type">Content type</Label>
            <select
              id="content-type"
              value={contentType}
              onChange={(e) =>
                setContentType(e.target.value as "json" | "form")
              }
              className="flex h-10 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
            >
              <option value="json">application/json</option>
              <option value="form">application/x-www-form-urlencoded</option>
            </select>
            {fieldErrors["config.content_type"] && (
              <p className="text-sm text-[#cf222e]">
                {fieldErrors["config.content_type"]}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="webhook-secret">
              Secret{" "}
              <span className="font-normal text-[#656d76]">(optional)</span>
            </Label>
            <Input
              id="webhook-secret"
              type="password"
              value={secret}
              onChange={(e) => setSecret(e.target.value)}
              placeholder="unchanged — enter new value to update"
              autoComplete="new-password"
            />
            {fieldErrors["config.secret"] && (
              <p className="text-sm text-[#cf222e]">
                {fieldErrors["config.secret"]}
              </p>
            )}
          </div>

          <div className="space-y-3">
            <Label>Events</Label>
            <label className="flex cursor-pointer items-center gap-2 rounded-md border border-[#d0d7de] bg-[#f6f8fa] p-3 text-sm font-semibold">
              <input
                type="checkbox"
                checked={sendEverything}
                onChange={(e) => handleSendEverythingChange(e.target.checked)}
              />
              Send me everything
            </label>
            <div className="grid grid-cols-2 gap-2 rounded-md border border-[#d0d7de] bg-[#f6f8fa] p-3">
              {INDIVIDUAL_EVENTS.map((eventName) => (
                <label
                  key={eventName}
                  className={`flex items-center gap-2 text-sm ${
                    sendEverything
                      ? "cursor-not-allowed opacity-50"
                      : "cursor-pointer"
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={selectedEvents.has(eventName)}
                    onChange={() => toggleEvent(eventName)}
                    disabled={sendEverything}
                  />
                  <span className="font-mono">{eventName}</span>
                </label>
              ))}
            </div>
            {fieldErrors.events && (
              <p className="text-sm text-[#cf222e]">{fieldErrors.events}</p>
            )}
          </div>

          <div className="flex items-center justify-between rounded-md border border-[#d0d7de] bg-[#f6f8fa] p-3">
            <div>
              <div className="text-sm font-semibold">Active</div>
              <p className="text-xs text-[#656d76]">
                We will deliver event details when this hook is active.
              </p>
            </div>
            <label className="inline-flex cursor-pointer items-center gap-2">
              <input
                type="checkbox"
                checked={active}
                onChange={(e) => setActive(e.target.checked)}
                className="peer sr-only"
              />
              <span
                className={`relative inline-block h-5 w-10 rounded-full transition-colors ${
                  active ? "bg-[#1f883d]" : "bg-[#d0d7de]"
                }`}
              >
                <span
                  className={`absolute top-0.5 left-0.5 h-4 w-4 rounded-full bg-white transition-transform ${
                    active ? "translate-x-5" : ""
                  }`}
                />
              </span>
            </label>
          </div>

          {formError && (
            <p className="text-sm text-[#cf222e]" role="alert">
              {formError}
            </p>
          )}

          <div className="flex justify-end gap-2">
            <Button variant="outline" asChild>
              <Link href={`/${owner}/${repo}/settings/hooks`}>Cancel</Link>
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? "Saving…" : "Update webhook"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
