"use client";

import { FormEvent, useEffect, useRef, useState } from "react";
import Link from "next/link";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";

type Step = 1 | 2 | 3;

const IMPORT_TYPES = [
  { key: "repositories", label: "Repositories", desc: "Branches, tags, commits" },
  { key: "issues", label: "Issues", desc: "Issues, labels, comments" },
  { key: "pull_requests", label: "Pull Requests", desc: "PRs, reviews, comments" },
  { key: "wiki", label: "Wiki", desc: "Wiki pages and revisions" },
] as const;

type ImportTypeKey = (typeof IMPORT_TYPES)[number]["key"];

interface ImportRun {
  run_id: string;
  status: "queued" | "running" | "completed" | "failed";
  progress?: {
    percentage: number;
    current_item?: string;
  };
  failed_items?: { id: string; reason: string }[];
}

export default function ImportWizardPage() {
  const [step, setStep] = useState<Step>(1);
  const [sourceUrl, setSourceUrl] = useState("");
  const [token, setToken] = useState("");
  const [urlError, setUrlError] = useState<string | null>(null);
  const [types, setTypes] = useState<Set<ImportTypeKey>>(
    new Set<ImportTypeKey>(["repositories", "issues", "pull_requests"]),
  );
  const [runId, setRunId] = useState<string | null>(null);
  const [run, setRun] = useState<ImportRun | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  const validateUrl = (raw: string): string | null => {
    const trimmed = raw.trim();
    if (!trimmed) return "Source URL is required.";
    try {
      const u = new URL(trimmed);
      if (!/^https?:$/.test(u.protocol)) {
        return "URL must use http or https.";
      }
      if (!/github\.com$/i.test(u.hostname) && u.hostname !== "github.com") {
        if (!u.hostname.endsWith("github.com")) {
          return "URL must point to a GitHub repository.";
        }
      }
      if (u.pathname.split("/").filter(Boolean).length < 2) {
        return "URL must include /owner/repo.";
      }
      return null;
    } catch {
      return "URL is not valid.";
    }
  };

  const goToStep2 = (e: FormEvent) => {
    e.preventDefault();
    const err = validateUrl(sourceUrl);
    setUrlError(err);
    if (err) return;
    if (!token.trim()) {
      setUrlError("Personal access token is required.");
      return;
    }
    setStep(2);
  };

  const startImport = async () => {
    setSubmitError(null);
    if (types.size === 0) {
      setSubmitError("Select at least one import type.");
      return;
    }
    setSubmitting(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/import`, {
        method: "POST",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          source_url: sourceUrl.trim(),
          token: token.trim(),
          types: Array.from(types),
        }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(
          (body as { message?: string }).message ??
            `Failed to start import (${res.status})`,
        );
      }
      const created = (await res.json()) as { run_id: string };
      setRunId(created.run_id);
      setStep(3);
      pollRun(created.run_id);
    } catch (err) {
      setSubmitError(
        err instanceof Error ? err.message : "Failed to start import.",
      );
    } finally {
      setSubmitting(false);
    }
  };

  const pollRun = (id: string) => {
    if (pollRef.current) clearInterval(pollRef.current);
    const tick = async () => {
      try {
        const res = await fetch(`${API_BASE}/api/v1/import/${id}`, {
          headers: { Accept: "application/json" },
          cache: "no-store",
        });
        if (!res.ok) return;
        const data = (await res.json()) as ImportRun;
        setRun(data);
        if (data.status === "completed" || data.status === "failed") {
          if (pollRef.current) {
            clearInterval(pollRef.current);
            pollRef.current = null;
          }
        }
      } catch {
        // ignore transient errors
      }
    };
    void tick();
    pollRef.current = setInterval(tick, 3000);
  };

  const toggleType = (key: ImportTypeKey) => {
    setTypes((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const percentage = run?.progress?.percentage ?? 0;

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
          href="/dashboard"
          className="rounded-md px-3 py-1.5 text-sm hover:bg-[#f6f8fa]"
        >
          Cancel
        </Link>
      </header>

      <div className="mx-auto max-w-[720px] px-6 py-8">
        <div className="mb-6 text-center">
          <h1 className="text-2xl font-semibold">Import from GitHub</h1>
          <p className="mt-1 text-sm text-[#656d76]">
            Migrate an existing GitHub repository in three steps.
          </p>
        </div>

        <Stepper step={step} />

        <div className="mt-6 rounded-md border border-[#d0d7de] bg-white">
          {step === 1 && (
            <form onSubmit={goToStep2} className="space-y-4 p-5">
              <h2 className="text-lg font-semibold">Source</h2>
              <div>
                <label
                  htmlFor="source-url"
                  className="mb-1.5 block text-sm font-semibold"
                >
                  GitHub repository URL{" "}
                  <span className="text-[#cf222e]">*</span>
                </label>
                <input
                  id="source-url"
                  type="url"
                  value={sourceUrl}
                  onChange={(e) => setSourceUrl(e.target.value)}
                  placeholder="https://github.com/owner/repo"
                  className="w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm font-mono"
                  required
                />
              </div>
              <div>
                <label
                  htmlFor="source-token"
                  className="mb-1.5 block text-sm font-semibold"
                >
                  Personal access token{" "}
                  <span className="text-[#cf222e]">*</span>
                </label>
                <input
                  id="source-token"
                  type="password"
                  value={token}
                  onChange={(e) => setToken(e.target.value)}
                  placeholder="ghp_xxx..."
                  className="w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm font-mono"
                  autoComplete="off"
                  required
                />
                <p className="mt-1 text-xs text-[#656d76]">
                  Token must have <code>repo</code> scope. Not stored after
                  import completes.
                </p>
              </div>
              {urlError && (
                <p className="text-sm text-[#cf222e]">{urlError}</p>
              )}
              <div className="flex justify-end">
                <button
                  type="submit"
                  className="rounded-md bg-[#0969da] px-4 py-2 text-sm font-semibold text-white hover:bg-[#0860ca]"
                >
                  Next →
                </button>
              </div>
            </form>
          )}

          {step === 2 && (
            <div className="space-y-4 p-5">
              <h2 className="text-lg font-semibold">What to import</h2>
              <div className="space-y-2">
                {IMPORT_TYPES.map((t) => (
                  <label
                    key={t.key}
                    className="flex cursor-pointer items-start gap-3 rounded-md border border-[#d0d7de] p-3 hover:bg-[#f6f8fa]"
                  >
                    <input
                      type="checkbox"
                      checked={types.has(t.key)}
                      onChange={() => toggleType(t.key)}
                      className="mt-1"
                    />
                    <div>
                      <div className="text-sm font-semibold">{t.label}</div>
                      <div className="text-xs text-[#656d76]">{t.desc}</div>
                    </div>
                  </label>
                ))}
              </div>
              {submitError && (
                <p className="text-sm text-[#cf222e]">{submitError}</p>
              )}
              <div className="flex justify-between">
                <button
                  type="button"
                  onClick={() => setStep(1)}
                  className="rounded-md border border-[#d0d7de] bg-white px-4 py-2 text-sm hover:bg-[#f6f8fa]"
                >
                  ← Back
                </button>
                <button
                  type="button"
                  onClick={startImport}
                  disabled={submitting || types.size === 0}
                  className="rounded-md bg-[#1f883d] px-4 py-2 text-sm font-semibold text-white hover:bg-[#1a7f37] disabled:opacity-50"
                >
                  {submitting ? "Starting…" : "Start import"}
                </button>
              </div>
            </div>
          )}

          {step === 3 && (
            <div className="space-y-4 p-5">
              <h2 className="text-lg font-semibold">Import progress</h2>
              {runId && (
                <p className="text-xs text-[#656d76]">
                  Run ID: <code className="font-mono">{runId}</code>
                </p>
              )}
              <div className="h-2 w-full overflow-hidden rounded-full bg-[#eaeef2]">
                <div
                  className="h-full bg-[#0969da] transition-all"
                  style={{
                    width: `${Math.max(0, Math.min(100, percentage))}%`,
                  }}
                />
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="font-semibold">
                  {Math.round(percentage)}% complete
                </span>
                <span className="text-[#656d76]">
                  Status: {run?.status ?? "queued"}
                </span>
              </div>
              {run?.progress?.current_item && (
                <p className="text-xs text-[#656d76]">
                  Currently importing:{" "}
                  <code className="font-mono">
                    {run.progress.current_item}
                  </code>
                </p>
              )}
              {(run?.status === "completed" || run?.status === "failed") && (
                <div className="rounded-md border border-[#d0d7de] bg-[#f6f8fa] p-3">
                  <div className="text-sm font-semibold">
                    {run.status === "completed"
                      ? "Import finished"
                      : "Import failed"}
                  </div>
                  {run.failed_items && run.failed_items.length > 0 ? (
                    <ul className="mt-2 space-y-1 text-xs">
                      {run.failed_items.map((item) => (
                        <li
                          key={item.id}
                          className="font-mono text-[#cf222e]"
                        >
                          {item.id}: {item.reason}
                        </li>
                      ))}
                    </ul>
                  ) : (
                    <p className="mt-1 text-xs text-[#656d76]">
                      No failed items.
                    </p>
                  )}
                </div>
              )}
              <div className="flex justify-end">
                <Link
                  href="/dashboard"
                  className="rounded-md border border-[#d0d7de] bg-white px-4 py-2 text-sm hover:bg-[#f6f8fa]"
                >
                  Back to dashboard
                </Link>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function Stepper({ step }: { step: Step }) {
  const items: { num: Step; label: string }[] = [
    { num: 1, label: "Source" },
    { num: 2, label: "Scope" },
    { num: 3, label: "Import" },
  ];
  return (
    <ol className="flex items-center justify-center gap-4">
      {items.map((item, idx) => {
        const status =
          step > item.num ? "done" : step === item.num ? "active" : "pending";
        return (
          <li key={item.num} className="flex items-center gap-2">
            <span
              className={`flex h-8 w-8 items-center justify-center rounded-full text-xs font-bold ${
                status === "done"
                  ? "bg-[#1f883d] text-white"
                  : status === "active"
                    ? "bg-[#0969da] text-white"
                    : "bg-[#eaeef2] text-[#656d76]"
              }`}
            >
              {status === "done" ? "✓" : item.num}
            </span>
            <span
              className={`text-sm ${
                status === "active"
                  ? "font-semibold text-[#0969da]"
                  : status === "done"
                    ? "text-[#1f883d]"
                    : "text-[#656d76]"
              }`}
            >
              {item.label}
            </span>
            {idx < items.length - 1 && (
              <span className="mx-2 inline-block h-px w-12 bg-[#d0d7de]" />
            )}
          </li>
        );
      })}
    </ol>
  );
}
