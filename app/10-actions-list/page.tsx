"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import StatusBadge from "@/components/StatusBadge";
import { useAuth } from "@/lib/auth";
import { env } from "@/lib/env";
import type { WorkflowRun, WorkflowRunsResponse } from "@/types/workflow";

function formatDuration(run: WorkflowRun): string {
  const start = run.started_at ?? run.run_started_at;
  if (!start) return "—";
  const startMs = new Date(start).getTime();
  const endMs = run.completed_at
    ? new Date(run.completed_at).getTime()
    : run.updated_at
      ? new Date(run.updated_at).getTime()
      : Date.now();
  const seconds = Math.max(0, Math.floor((endMs - startMs) / 1000));
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}m ${s}s`;
}

function formatCreatedAt(createdAt?: string): string {
  if (!createdAt) return "—";
  return new Date(createdAt).toLocaleString();
}

function SkeletonRows() {
  return (
    <>
      {Array.from({ length: 5 }).map((_, index) => (
        <tr key={index} data-testid="run-skeleton-row">
          {Array.from({ length: 8 }).map((__, cellIndex) => (
            <td key={cellIndex} className="px-4 py-3">
              <div className="h-4 bg-gray-200 rounded animate-pulse" />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

export default function Page() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { token } = useAuth();

  const owner = searchParams.get("owner") ?? "";
  const repo = searchParams.get("repo") ?? "";

  const [runs, setRuns] = useState<WorkflowRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState("");
  const [branchFilter, setBranchFilter] = useState("");

  const loadRuns = useCallback(async () => {
    if (!owner || !repo) {
      setLoading(false);
      setError("Missing owner or repo");
      return;
    }

    setLoading(true);
    setError(null);

    const params = new URLSearchParams();
    params.set("status", statusFilter);
    params.set("branch", branchFilter);

    const url = `${env.NEXT_PUBLIC_API_BASE_URL}/repos/${owner}/${repo}/actions/runs?${params.toString()}`;

    try {
      const headers: Record<string, string> = {
        Accept: "application/vnd.github+json",
      };
      if (token) {
        headers.Authorization = `Bearer ${token}`;
      }

      const res = await fetch(url, { headers });
      if (!res.ok) throw new Error("Failed to load workflow runs");
      const data = (await res.json()) as WorkflowRunsResponse;
      setRuns(data.workflow_runs ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load workflow runs");
    } finally {
      setLoading(false);
    }
  }, [owner, repo, statusFilter, branchFilter, token]);

  useEffect(() => {
    loadRuns();
  }, [loadRuns]);

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <header className="sticky top-0 z-50 h-16 flex items-center justify-between px-6 bg-white/85 backdrop-blur border-b border-[var(--border)]">
        <div className="flex items-center gap-3 font-extrabold text-lg">
          <span>🐙</span>
          <strong>OpenHub</strong>
        </div>
        <div className="flex items-center gap-4">
          <Link
            href="/07-repo-detail"
            className="px-3 py-1.5 text-sm rounded-md border border-[#d0d7de] hover:bg-[#f3f4f6]"
          >
            ← リポジトリへ戻る
          </Link>
        </div>
      </header>

      <div className="bg-white border-b border-[#d0d7de] px-6 py-4">
        <div className="text-xl font-semibold">
          <Link href="/07-repo-detail" className="text-[#0969da] hover:underline">
            {owner || "openhub"}
          </Link>
          {" / "}
          <Link href="/07-repo-detail" className="text-[#0969da] hover:underline">
            <strong>{repo || "awesome-project"}</strong>
          </Link>
          <span className="ml-2 inline-block px-2 py-0.5 text-xs rounded-full border border-[#d0d7de] text-[#656d76] align-middle">
            Public
          </span>
        </div>
        <nav className="flex gap-1 mt-4">
          <Link href="/07-repo-detail" className="px-4 py-2 text-sm rounded-t-md flex items-center gap-1.5 hover:bg-[#f3f4f6]">
            📄 Code
          </Link>
          <Link href="/08-issues-list" className="px-4 py-2 text-sm rounded-t-md flex items-center gap-1.5 hover:bg-[#f3f4f6]">
            ⊙ Issues
          </Link>
          <Link href="/09-pr-list" className="px-4 py-2 text-sm rounded-t-md flex items-center gap-1.5 hover:bg-[#f3f4f6]">
            ⇆ Pull requests
          </Link>
          <Link
            href="/10-actions-list"
            className="px-4 py-2 text-sm rounded-t-md flex items-center gap-1.5 border-b-2 border-[#fd8c73] font-semibold"
          >
            ▶ Actions
          </Link>
        </nav>
      </div>

      <div className="max-w-[1280px] mx-auto px-6 py-6">
        {error && (
          <div className="mb-4 rounded-md border border-[#cf222e] bg-[#ffebe9] px-4 py-3 text-sm text-[#cf222e]">
            {error}
          </div>
        )}

        <div className="flex gap-2 items-center mb-4 flex-wrap">
          <select
            aria-label="Status filter"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-[#f6f8fa]"
          >
            <option value="">All statuses</option>
            <option value="queued">Queued</option>
            <option value="in_progress">In progress</option>
            <option value="completed">Completed</option>
          </select>
          <input
            aria-label="Branch filter"
            type="text"
            value={branchFilter}
            onChange={(e) => setBranchFilter(e.target.value)}
            placeholder="Filter by branch"
            className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md min-w-[200px]"
          />
        </div>

        <div className="bg-white border border-[#d0d7de] rounded-md overflow-hidden">
          <div className="px-4 py-3 bg-[#f6f8fa] border-b border-[#d0d7de] text-sm text-[#656d76]">
            {loading ? (
              "Loading…"
            ) : (
              <span>
                <strong>{runs.length}</strong> workflow runs
              </span>
            )}
          </div>

          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[#d0d7de] bg-[#f6f8fa] text-left text-xs text-[#656d76]">
                <th className="px-4 py-2 font-medium">Status</th>
                <th className="px-4 py-2 font-medium">Run</th>
                <th className="px-4 py-2 font-medium">Workflow</th>
                <th className="px-4 py-2 font-medium">Event</th>
                <th className="px-4 py-2 font-medium">Branch</th>
                <th className="px-4 py-2 font-medium">SHA</th>
                <th className="px-4 py-2 font-medium">Duration</th>
                <th className="px-4 py-2 font-medium">Created</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <SkeletonRows />
              ) : runs.length === 0 ? (
                <tr>
                  <td colSpan={8} className="px-4 py-8 text-center text-[#656d76]">
                    No workflow runs found.
                  </td>
                </tr>
              ) : (
                runs.map((run) => (
                  <tr
                    key={run.id}
                    onClick={() =>
                      router.push(`/${owner}/${repo}/actions/runs/${run.id}`)
                    }
                    className="border-b border-[#d8dee4] last:border-b-0 hover:bg-[#fafbfc] cursor-pointer"
                  >
                    <td className="px-4 py-3">
                      <StatusBadge
                        status={run.status}
                        conclusion={run.conclusion ?? undefined}
                      />
                    </td>
                    <td className="px-4 py-3 font-mono text-xs">#{run.run_number}</td>
                    <td className="px-4 py-3 font-semibold">{run.name}</td>
                    <td className="px-4 py-3 text-[#656d76]">{run.event ?? "—"}</td>
                    <td className="px-4 py-3">
                      <span className="bg-[#ddf4ff] text-[#0969da] px-1.5 py-0.5 rounded font-mono text-[11px]">
                        {run.head_branch}
                      </span>
                    </td>
                    <td className="px-4 py-3 font-mono text-xs">
                      {run.head_sha.slice(0, 7)}
                    </td>
                    <td className="px-4 py-3 text-[#656d76]">{formatDuration(run)}</td>
                    <td className="px-4 py-3 text-[#656d76]">
                      {formatCreatedAt(run.created_at)}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
