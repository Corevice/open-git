"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { apiClient, isApiError } from "@/lib/api-client";

export interface BranchItem {
  name: string;
  commit: { sha: string };
}

const BRANCH_NAME_PATTERN = /^(?!.*\.\.)[a-zA-Z0-9._\/-]+$/;

interface CreateBranchFormProps {
  owner: string;
  repo: string;
  branches: BranchItem[];
}

export default function CreateBranchForm({
  owner,
  repo,
  branches,
}: CreateBranchFormProps) {
  const router = useRouter();
  const [branchName, setBranchName] = useState("");
  const [fromRef, setFromRef] = useState(branches[0]?.name ?? "");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    const trimmed = branchName.trim();
    if (!trimmed) {
      setError("Branch name is required");
      return;
    }
    if (!BRANCH_NAME_PATTERN.test(trimmed)) {
      setError("Branch name contains invalid characters");
      return;
    }
    if (branches.length === 0) {
      setError("No source branches available");
      return;
    }

    const source = branches.find((b) => b.name === fromRef);
    if (!source) {
      setError("Select a source branch");
      return;
    }

    setLoading(true);
    try {
      await apiClient.createRef(
        owner,
        repo,
        `refs/heads/${trimmed}`,
        source.commit.sha,
      );
      setBranchName("");
      router.refresh();
    } catch (err) {
      if (isApiError(err)) {
        setError(err.message);
      } else {
        setError("Failed to create branch");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="bg-white border border-[#d0d7de] rounded-lg p-4 mb-6"
    >
      <h2 className="text-sm font-semibold m-0 mb-3">Create a branch</h2>
      {error && (
        <div
          role="alert"
          className="mb-3 px-3 py-2 text-sm text-red-800 bg-red-50 border border-red-200 rounded-md"
        >
          {error}
        </div>
      )}
      <div className="flex flex-wrap items-end gap-3">
        <div>
          <label htmlFor="branch-name" className="block text-sm mb-1">
            Branch name
          </label>
          <input
            id="branch-name"
            type="text"
            value={branchName}
            onChange={(e) => setBranchName(e.target.value)}
            className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md min-w-[200px]"
            placeholder="new-branch"
            disabled={loading}
          />
        </div>
        <div>
          <label htmlFor="from-ref" className="block text-sm mb-1">
            From
          </label>
          <select
            id="from-ref"
            value={fromRef}
            onChange={(e) => setFromRef(e.target.value)}
            className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md"
            disabled={loading || branches.length === 0}
          >
            {branches.map((branch) => (
              <option key={branch.name} value={branch.name}>
                {branch.name}
              </option>
            ))}
          </select>
        </div>
        <button
          type="submit"
          disabled={loading || branches.length === 0}
          className="px-4 py-1.5 text-sm font-semibold text-white bg-[#1f883d] border border-black/10 rounded-md hover:bg-[#1a7f37] disabled:opacity-50"
        >
          {loading ? "Creating…" : "Create branch"}
        </button>
      </div>
    </form>
  );
}

interface BranchDeleteButtonProps {
  owner: string;
  repo: string;
  branch: string;
  disabled?: boolean;
}

export function BranchDeleteButton({
  owner,
  repo,
  branch,
  disabled = false,
}: BranchDeleteButtonProps) {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleDelete() {
    if (disabled || loading) return;
    if (
      !window.confirm(
        `Delete branch "${branch}"? This action cannot be undone.`,
      )
    ) {
      return;
    }
    setError(null);
    setLoading(true);
    try {
      await apiClient.deleteBranch(owner, repo, branch);
      router.refresh();
    } catch (err) {
      if (isApiError(err)) {
        setError(err.message);
      } else {
        setError("Failed to delete branch");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex flex-col items-end gap-1">
      {error && (
        <div
          role="alert"
          className="px-2 py-1 text-xs text-red-800 bg-red-50 border border-red-200 rounded-md"
        >
          {error}
        </div>
      )}
      <button
        type="button"
        onClick={handleDelete}
        disabled={disabled || loading}
        className="px-3 py-1 text-sm text-red-600 border border-[#d0d7de] rounded-md hover:bg-red-50 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {loading ? "Deleting…" : "Delete"}
      </button>
    </div>
  );
}
