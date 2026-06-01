"use client";

import { useState } from "react";
import type { PullRequest } from "@/lib/api-types";
import { ApiError } from "@/lib/api";

type MergePanelProps = {
  pr: PullRequest;
  onMerge: (method: string) => Promise<void>;
};

const MERGE_METHODS = [
  { value: "merge", label: "Create a merge commit" },
  { value: "squash", label: "Squash and merge" },
  { value: "rebase", label: "Rebase and merge" },
];

function hasConflicts(pr: PullRequest): boolean {
  return pr.state === "conflict" || pr.state === "conflicts";
}

function isMerged(pr: PullRequest): boolean {
  return pr.mergedAt != null || pr.state === "merged";
}

export default function MergePanel({ pr, onMerge }: MergePanelProps) {
  const [method, setMethod] = useState("merge");
  const [merging, setMerging] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const merged = isMerged(pr);
  const conflicts = hasConflicts(pr);
  const blocked = conflicts;

  const handleMerge = async () => {
    setMerging(true);
    setError(null);
    try {
      await onMerge(method);
    } catch (e) {
      if (e instanceof ApiError) {
        if (e.status === 405) {
          setError(e.message);
        } else if (e.status === 409) {
          setError("Merge conflict — please resolve before merging");
        } else {
          setError(e.message);
        }
      } else {
        setError(e instanceof Error ? e.message : "Failed to merge pull request");
      }
    } finally {
      setMerging(false);
    }
  };

  if (merged) {
    return (
      <div className="border border-[#d0d7de] rounded-lg p-4 mt-4 bg-[#f6f8fa] flex items-center gap-3">
        <span className="inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-semibold text-white bg-[#8250df]">
          Merged
        </span>
        <span className="text-sm text-[#656d76]">
          This pull request has already been merged
          {pr.mergedAt ? ` on ${new Date(pr.mergedAt).toLocaleDateString()}` : ""}.
        </span>
      </div>
    );
  }

  const blockedTooltip = conflicts
    ? "Resolve merge conflicts before merging"
    : undefined;

  return (
    <div className="border border-[#d0d7de] rounded-lg p-4 mt-4 bg-white">
      <div className="flex items-start gap-4">
        <div className={`text-2xl ${conflicts ? "text-[#cf222e]" : "text-[#1a7f37]"}`}>
          {conflicts ? "✕" : "✓"}
        </div>
        <div className="flex-1">
          {conflicts ? (
            <>
              <strong className="block text-[#cf222e]">This branch has conflicts</strong>
              <span className="text-sm text-[#656d76]">
                Resolve conflicts before merging this pull request.
              </span>
            </>
          ) : (
            <>
              <strong className="block text-[#1a7f37]">This branch has no conflicts</strong>
              <span className="text-sm text-[#656d76]">
                Merging can be performed automatically.
              </span>
            </>
          )}
        </div>
      </div>

      {error && (
        <p className="mt-3 text-sm text-[#cf222e] bg-[#ffebe9] border border-[#ff8182] rounded-md px-3 py-2">
          {error}
        </p>
      )}

      <div className="mt-4 flex flex-wrap items-center gap-3">
        <select
          value={method}
          onChange={(e) => setMethod(e.target.value)}
          disabled={blocked || merging}
          className="px-3 py-2 text-sm border border-[#d0d7de] rounded-md bg-white disabled:opacity-50"
        >
          {MERGE_METHODS.map((m) => (
            <option key={m.value} value={m.value}>
              {m.label}
            </option>
          ))}
        </select>

        <span title={blockedTooltip}>
          <button
            type="button"
            onClick={handleMerge}
            disabled={blocked || merging}
            className="px-4 py-2 bg-[#1f883d] text-white rounded-md text-sm font-semibold border border-black/10 hover:bg-[#1a7f37] disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {merging ? "Merging…" : "Merge pull request"}
          </button>
        </span>
      </div>
    </div>
  );
}
