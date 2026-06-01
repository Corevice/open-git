"use client";

import { FormEvent, useState } from "react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";

export const WEBHOOK_EVENTS = [
  "push",
  "issues",
  "pull_request",
  "create",
  "delete",
  "workflow_run",
] as const;

export type WebhookEvent = (typeof WEBHOOK_EVENTS)[number];

export interface WebhookInitial {
  id?: number;
  url?: string;
  events?: string[];
  active?: boolean;
}

type WebhookFormProps = {
  owner: string;
  repo: string;
  initial?: WebhookInitial;
  mode?: "create" | "edit";
  onSaved?: () => void;
};

export default function WebhookForm({
  owner,
  repo,
  initial,
  mode = "create",
  onSaved,
}: WebhookFormProps) {
  const [url, setUrl] = useState(initial?.url ?? "");
  const [secret, setSecret] = useState("");
  const [events, setEvents] = useState<Set<string>>(
    new Set(initial?.events ?? ["push"]),
  );
  const [active, setActive] = useState<boolean>(initial?.active ?? true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const toggleEvent = (event: string) => {
    setEvents((prev) => {
      const next = new Set(prev);
      if (next.has(event)) {
        next.delete(event);
      } else {
        next.add(event);
      }
      return next;
    });
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    const trimmedUrl = url.trim();
    if (!trimmedUrl) {
      setError("URL is required.");
      return;
    }
    try {
      new URL(trimmedUrl);
    } catch {
      setError("URL is invalid.");
      return;
    }
    if (events.size === 0) {
      setError("Select at least one event.");
      return;
    }

    const config: Record<string, unknown> = {
      url: trimmedUrl,
      content_type: "json",
    };
    if (secret.trim()) {
      config.secret = secret.trim();
    }

    const payload = {
      config,
      events: Array.from(events),
      active,
    };

    const url2 =
      mode === "edit" && initial?.id
        ? `${API_BASE}/repos/${owner}/${repo}/hooks/${initial.id}`
        : `${API_BASE}/repos/${owner}/${repo}/hooks`;
    const method = mode === "edit" ? "PATCH" : "POST";

    setSubmitting(true);
    try {
      const res = await fetch(url2, {
        method,
        headers: {
          Accept: "application/vnd.github+json",
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(
          (body as { message?: string }).message ??
            `Failed to save webhook (${res.status})`,
        );
      }

      setSuccess(mode === "edit" ? "Webhook updated." : "Webhook created.");
      if (mode === "create") {
        setUrl("");
        setSecret("");
        setEvents(new Set(["push"]));
        setActive(true);
      }
      onSaved?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save webhook.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="space-y-4 rounded-md border border-[#d0d7de] bg-white p-5"
    >
      <div>
        <label
          htmlFor="wh-url"
          className="mb-1.5 block text-sm font-semibold"
        >
          Payload URL <span className="text-[#cf222e]">*</span>
        </label>
        <input
          id="wh-url"
          type="url"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://example.com/webhook"
          className="w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm font-mono"
          required
        />
      </div>

      <div>
        <label
          htmlFor="wh-secret"
          className="mb-1.5 block text-sm font-semibold"
        >
          Secret <span className="text-[#656d76] font-normal">(optional)</span>
        </label>
        <input
          id="wh-secret"
          type="password"
          value={secret}
          onChange={(e) => setSecret(e.target.value)}
          placeholder={
            mode === "edit"
              ? "Leave blank to keep current secret"
              : "Used to sign payloads with HMAC-SHA256"
          }
          className="w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm font-mono"
          autoComplete="new-password"
        />
        <p className="mt-1 text-xs text-[#656d76]">
          Used to compute the <code>X-Hub-Signature-256</code> header.
        </p>
      </div>

      <div>
        <div className="mb-2 text-sm font-semibold">Events</div>
        <div className="grid grid-cols-2 gap-2 rounded-md border border-[#d0d7de] bg-[#f6f8fa] p-3">
          {WEBHOOK_EVENTS.map((event) => (
            <label
              key={event}
              className="flex cursor-pointer items-center gap-2 text-sm"
            >
              <input
                type="checkbox"
                checked={events.has(event)}
                onChange={() => toggleEvent(event)}
              />
              <span className="font-mono">{event}</span>
            </label>
          ))}
        </div>
      </div>

      <div className="flex items-center justify-between rounded-md border border-[#d0d7de] bg-[#f6f8fa] p-3">
        <div>
          <div className="text-sm font-semibold">Active</div>
          <p className="text-xs text-[#656d76]">
            Only deliver events while the webhook is active.
          </p>
        </div>
        <label className="inline-flex cursor-pointer items-center gap-2">
          <input
            type="checkbox"
            checked={active}
            onChange={(e) => setActive(e.target.checked)}
            className="sr-only peer"
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
          <span className="text-xs text-[#656d76]">
            {active ? "Active" : "Inactive"}
          </span>
        </label>
      </div>

      {error && (
        <p className="text-sm text-[#cf222e]" role="alert">
          {error}
        </p>
      )}
      {success && (
        <p className="text-sm text-[#1f883d]" role="status">
          {success}
        </p>
      )}

      <div className="flex justify-end">
        <button
          type="submit"
          disabled={submitting}
          className="rounded-md bg-[#1f883d] px-4 py-2 text-sm font-semibold text-white hover:bg-[#1a7f37] disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {submitting
            ? "Saving…"
            : mode === "edit"
              ? "Update webhook"
              : "Add webhook"}
        </button>
      </div>
    </form>
  );
}
