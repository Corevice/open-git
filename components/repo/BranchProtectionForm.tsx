"use client";

import { FormEvent, useState } from "react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";

export interface BranchProtectionInitial {
  pattern?: string;
  required_reviews?: number;
  required_checks?: string[];
  force_push_blocked?: boolean;
}

type BranchProtectionFormProps = {
  owner: string;
  repo: string;
  initial?: BranchProtectionInitial;
  mode?: "create" | "edit";
  onSaved?: () => void;
};

export default function BranchProtectionForm({
  owner,
  repo,
  initial,
  mode = "create",
  onSaved,
}: BranchProtectionFormProps) {
  const [pattern, setPattern] = useState(initial?.pattern ?? "");
  const [requiredReviews, setRequiredReviews] = useState<number>(
    initial?.required_reviews ?? 0,
  );
  const [requiredChecksText, setRequiredChecksText] = useState<string>(
    (initial?.required_checks ?? []).join("\n"),
  );
  const [forcePushBlocked, setForcePushBlocked] = useState<boolean>(
    initial?.force_push_blocked ?? true,
  );
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    const trimmedPattern = pattern.trim();
    if (!trimmedPattern) {
      setError("Branch pattern is required.");
      return;
    }

    const reviewsNum = Number(requiredReviews);
    if (!Number.isInteger(reviewsNum) || reviewsNum < 0 || reviewsNum > 6) {
      setError("Required reviews must be an integer between 0 and 6.");
      return;
    }

    const requiredChecks = requiredChecksText
      .split("\n")
      .map((line) => line.trim())
      .filter((line) => line.length > 0);

    const payload = {
      pattern: trimmedPattern,
      required_reviews: reviewsNum,
      required_checks: requiredChecks,
      force_push_blocked: forcePushBlocked,
    };

    const method = mode === "edit" ? "PATCH" : "POST";
    const url = `${API_BASE}/repos/${owner}/${repo}/branches/${encodeURIComponent(trimmedPattern)}/protection`;

    setSubmitting(true);
    try {
      const res = await fetch(url, {
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
            `Failed to save branch protection (${res.status})`,
        );
      }

      setSuccess(
        mode === "edit"
          ? "Branch protection updated."
          : "Branch protection created.",
      );
      onSaved?.();
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
      <div>
        <label
          htmlFor="bp-pattern"
          className="mb-1.5 block text-sm font-semibold"
        >
          Branch name pattern <span className="text-[#cf222e]">*</span>
        </label>
        <input
          id="bp-pattern"
          type="text"
          value={pattern}
          onChange={(e) => setPattern(e.target.value)}
          placeholder="main, release/*"
          className="w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm font-mono"
          disabled={mode === "edit"}
          required
        />
        <p className="mt-1 text-xs text-[#656d76]">
          Glob pattern. Use <code className="font-mono">*</code> as a wildcard.
        </p>
      </div>

      <div>
        <label
          htmlFor="bp-required-reviews"
          className="mb-1.5 block text-sm font-semibold"
        >
          Required approving reviews
        </label>
        <input
          id="bp-required-reviews"
          type="number"
          min={0}
          max={6}
          step={1}
          value={requiredReviews}
          onChange={(e) =>
            setRequiredReviews(
              e.target.value === "" ? 0 : parseInt(e.target.value, 10),
            )
          }
          className="w-24 rounded-md border border-[#d0d7de] px-3 py-2 text-sm"
        />
        <p className="mt-1 text-xs text-[#656d76]">
          Number of approving reviews required before a PR can be merged (0–6).
        </p>
      </div>

      <div>
        <label
          htmlFor="bp-required-checks"
          className="mb-1.5 block text-sm font-semibold"
        >
          Required status checks
        </label>
        <textarea
          id="bp-required-checks"
          value={requiredChecksText}
          onChange={(e) => setRequiredChecksText(e.target.value)}
          placeholder="ci/test&#10;ci/build"
          rows={4}
          className="w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm font-mono"
        />
        <p className="mt-1 text-xs text-[#656d76]">
          One check name per line. Leave empty to require none.
        </p>
      </div>

      <div className="flex items-center justify-between rounded-md border border-[#d0d7de] bg-[#f6f8fa] p-3">
        <div>
          <div className="text-sm font-semibold">Block force pushes</div>
          <p className="text-xs text-[#656d76]">
            Reject any push that rewrites history on matching branches.
          </p>
        </div>
        <label className="inline-flex cursor-pointer items-center gap-2">
          <input
            type="checkbox"
            checked={forcePushBlocked}
            onChange={(e) => setForcePushBlocked(e.target.checked)}
            className="sr-only peer"
          />
          <span
            className={`relative inline-block h-5 w-10 rounded-full transition-colors ${
              forcePushBlocked ? "bg-[#1f883d]" : "bg-[#d0d7de]"
            }`}
          >
            <span
              className={`absolute top-0.5 left-0.5 h-4 w-4 rounded-full bg-white transition-transform ${
                forcePushBlocked ? "translate-x-5" : ""
              }`}
            />
          </span>
          <span className="text-xs text-[#656d76]">
            {forcePushBlocked ? "Blocked" : "Allowed"}
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
              ? "Update rule"
              : "Create rule"}
        </button>
      </div>
    </form>
  );
}
