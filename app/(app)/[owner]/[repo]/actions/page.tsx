"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";

type WorkflowRun = {
  id: number;
  name: string;
  run_number: number;
  status: string;
  conclusion: string | null;
  head_branch: string;
  head_sha: string;
  run_started_at?: string;
  updated_at?: string;
  actor?: { login: string };
  triggering_actor?: { login: string };
  event?: string;
};

type WorkflowRunsResponse = {
  workflow_runs: WorkflowRun[];
  total_count?: number;
};

function statusBadgeClass(status: string, conclusion: string | null): string {
  if (status === "in_progress" || status === "queued" || status === "waiting") {
    return "bg-[#fff8c5] text-[#9a6700]";
  }
  if (conclusion === "success") return "bg-[#dafbe1] text-[#1a7f37]";
  if (conclusion === "failure") return "bg-[#ffebe9] text-[#cf222e]";
  if (conclusion === "cancelled") return "bg-[#eaeef2] text-[#656d76]";
  return "bg-[#eaeef2] text-[#656d76]";
}

function statusLabel(status: string, conclusion: string | null): string {
  if (status === "in_progress" || status === "queued") return "In progress";
  if (conclusion) return conclusion.charAt(0).toUpperCase() + conclusion.slice(1);
  return status;
}

function formatDuration(run: WorkflowRun): string {
  if (!run.run_started_at) return "—";
  const start = new Date(run.run_started_at).getTime();
  const end = run.updated_at ? new Date(run.updated_at).getTime() : Date.now();
  const seconds = Math.max(0, Math.floor((end - start) / 1000));
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}m ${s}s`;
}

function triggeredBy(run: WorkflowRun): string {
  const actor = run.triggering_actor?.login ?? run.actor?.login ?? "unknown";
  const event = run.event ?? "workflow";
  return `${event} by ${actor}`;
}

function isFailed(run: WorkflowRun): boolean {
  return run.conclusion === "failure";
}

export default function ActionsPage() {
  const params = useParams();
  const owner = params.owner as string;
  const repo = params.repo as string;

  const [runs, setRuns] = useState<WorkflowRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [rerunningId, setRerunningId] = useState<number | null>(null);

  const loadRuns = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/repos/${owner}/${repo}/actions/runs`);
      if (!res.ok) throw new Error("Failed to load workflow runs");
      const data = (await res.json()) as WorkflowRunsResponse;
      setRuns(data.workflow_runs ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load workflow runs");
    } finally {
      setLoading(false);
    }
  }, [owner, repo]);

  useEffect(() => {
    loadRuns();
  }, [loadRuns]);

  async function handleRerun(runId: number) {
    setRerunningId(runId);
    try {
      const res = await fetch(
        `/api/repos/${owner}/${repo}/actions/runs/${runId}/rerun`,
        { method: "POST" },
      );
      if (!res.ok) throw new Error("Failed to re-run workflow");
      await loadRuns();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to re-run workflow");
    } finally {
      setRerunningId(null);
    }
  }

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <div className="max-w-[1280px] mx-auto px-6 py-6">
        <h1 className="text-2xl font-semibold mb-6">
          <span className="text-[#0969da]">{owner}</span> /{" "}
          <span className="text-[#0969da]">{repo}</span>
          <span className="ml-2 text-lg font-normal text-[#656d76]">Actions</span>
        </h1>

        {error && (
          <div className="mb-4 rounded-md border border-[#cf222e] bg-[#ffebe9] px-4 py-3 text-sm text-[#cf222e]">
            {error}
          </div>
        )}

        <div className="bg-white border border-[#d0d7de] rounded-md overflow-hidden">
          <div className="px-4 py-3 bg-[#f6f8fa] border-b border-[#d0d7de] text-sm text-[#656d76]">
            {loading ? "Loading…" : (
              <span>
                <strong>{runs.length}</strong> workflow runs
              </span>
            )}
          </div>

          {loading ? (
            <div className="px-4 py-8 text-center text-[#656d76]">Loading workflow runs…</div>
          ) : runs.length === 0 ? (
            <div className="px-4 py-8 text-center text-[#656d76]">No workflow runs yet.</div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-[#d0d7de] bg-[#f6f8fa] text-left text-xs text-[#656d76]">
                  <th className="px-4 py-2 font-medium">Workflow</th>
                  <th className="px-4 py-2 font-medium">Run</th>
                  <th className="px-4 py-2 font-medium">Status</th>
                  <th className="px-4 py-2 font-medium">Branch</th>
                  <th className="px-4 py-2 font-medium">Commit</th>
                  <th className="px-4 py-2 font-medium">Duration</th>
                  <th className="px-4 py-2 font-medium">Triggered by</th>
                  <th className="px-4 py-2 font-medium w-24" />
                </tr>
              </thead>
              <tbody>
                {runs.map((run) => (
                  <tr
                    key={run.id}
                    className="border-b border-[#d8dee4] last:border-b-0 hover:bg-[#fafbfc]"
                  >
                    <td className="px-4 py-3">
                      <Link
                        href={`/${owner}/${repo}/actions/runs/${run.id}`}
                        className="font-semibold text-[#0969da] hover:underline"
                      >
                        {run.name}
                      </Link>
                    </td>
                    <td className="px-4 py-3 font-mono text-xs">#{run.run_number}</td>
                    <td className="px-4 py-3">
                      <span
                        className={`inline-block px-2 py-0.5 rounded-full text-xs ${statusBadgeClass(run.status, run.conclusion)}`}
                      >
                        {statusLabel(run.status, run.conclusion)}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="bg-[#ddf4ff] text-[#0969da] px-1.5 py-0.5 rounded font-mono text-[11px]">
                        {run.head_branch}
                      </span>
                    </td>
                    <td className="px-4 py-3 font-mono text-xs">
                      {run.head_sha.slice(0, 7)}
                    </td>
                    <td className="px-4 py-3 text-[#656d76]">{formatDuration(run)}</td>
                    <td className="px-4 py-3 text-[#656d76]">{triggeredBy(run)}</td>
                    <td className="px-4 py-3">
                      {isFailed(run) && (
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          disabled={rerunningId === run.id}
                          onClick={() => handleRerun(run.id)}
                        >
                          {rerunningId === run.id ? "Re-running…" : "Re-run"}
                        </Button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  );
}
