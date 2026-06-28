"use client";

import Link from "next/link";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

import { RunStatusBadge } from "@/components/actions/RunStatusBadge";
import { Button } from "@/components/ui/button";

type WorkflowRun = {
  id: number;
  name: string;
  run_number: number;
  status: string;
  conclusion: string | null;
  head_branch: string;
  head_sha: string;
  event?: string;
  created_at?: string;
  updated_at?: string;
  run_started_at?: string;
  actor?: { login: string };
  triggering_actor?: { login: string };
};

type WorkflowJob = {
  id: number;
  name: string;
  status: string;
  conclusion: string | null;
  runner_label?: string;
  labels?: string[];
  started_at?: string | null;
  completed_at?: string | null;
};

function runApiPath(owner: string, repo: string, runId: string, suffix = ""): string {
  return `/api/repos/${owner}/${repo}/actions/runs/${runId}${suffix}`;
}

function eventIcon(event?: string): string {
  switch (event) {
    case "push":
      return "⬆";
    case "pull_request":
      return "⇄";
    case "workflow_dispatch":
      return "▶";
    default:
      return "⚡";
  }
}

function formatTimestamp(value?: string): string {
  if (!value) return "—";
  return new Date(value).toLocaleString();
}

function formatDuration(start?: string | null, end?: string | null): string {
  if (!start) return "—";
  const startMs = new Date(start).getTime();
  const endMs = end ? new Date(end).getTime() : Date.now();
  const seconds = Math.max(0, Math.floor((endMs - startMs) / 1000));
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}m ${s}s`;
}

function runDuration(run: WorkflowRun): string {
  const start = run.run_started_at ?? run.created_at;
  const end = run.status === "completed" ? run.updated_at : undefined;
  return formatDuration(start, end);
}

function actorLogin(run: WorkflowRun): string {
  return run.triggering_actor?.login ?? run.actor?.login ?? "unknown";
}

function jobRunnerLabel(job: WorkflowJob): string {
  if (job.runner_label) return job.runner_label;
  if (job.labels?.length) return job.labels.join(", ");
  return "—";
}

function canCancelRun(run: WorkflowRun): boolean {
  return run.status === "in_progress" || run.status === "queued";
}

function canRerunRun(run: WorkflowRun): boolean {
  return run.status === "completed";
}

export default function ActionsRunDetailPage() {
  const searchParams = useSearchParams();
  const params = useParams();
  const router = useRouter();

  const owner = (searchParams.get("owner") ?? (params.owner as string | undefined) ?? "openhub") as string;
  const repo = (searchParams.get("repo") ?? (params.repo as string | undefined) ?? "awesome-project") as string;
  const runId = (searchParams.get("runId") ??
    (params.runId as string | undefined) ??
    (params.id as string | undefined) ??
    "1") as string;

  const [run, setRun] = useState<WorkflowRun | null>(null);
  const [jobs, setJobs] = useState<WorkflowJob[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  const loadRunDetail = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [runRes, jobsRes] = await Promise.all([
        fetch(runApiPath(owner, repo, runId)),
        fetch(runApiPath(owner, repo, runId, "/jobs")),
      ]);

      if (!runRes.ok) throw new Error("Failed to load workflow run");
      if (!jobsRes.ok) throw new Error("Failed to load workflow jobs");

      const runData = (await runRes.json()) as WorkflowRun;
      const jobsData = (await jobsRes.json()) as { jobs?: WorkflowJob[] };
      setRun(runData);
      setJobs(jobsData.jobs ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load workflow run");
      setRun(null);
      setJobs([]);
    } finally {
      setLoading(false);
    }
  }, [owner, repo, runId]);

  useEffect(() => {
    loadRunDetail();
  }, [loadRunDetail]);

  async function handleCancel() {
    if (!run) return;
    setActionLoading(true);
    setError(null);
    try {
      const res = await fetch(runApiPath(owner, repo, runId, "/cancel"), {
        method: "POST",
      });
      if (!res.ok) throw new Error("Failed to cancel workflow run");
      await loadRunDetail();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to cancel workflow run");
    } finally {
      setActionLoading(false);
    }
  }

  async function handleRerun() {
    if (!run) return;
    setActionLoading(true);
    setError(null);
    try {
      const res = await fetch(runApiPath(owner, repo, runId, "/rerun"), {
        method: "POST",
      });
      if (!res.ok) throw new Error("Failed to re-run workflow");
      router.push(`/11-actions-run?owner=${owner}&repo=${repo}&runId=${runId}`);
      await loadRunDetail();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to re-run workflow");
    } finally {
      setActionLoading(false);
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-[#f6f8fa] px-6 py-8 text-[#656d76]">
        Loading workflow run…
      </div>
    );
  }

  if (error && !run) {
    return (
      <div className="min-h-screen bg-[#f6f8fa] px-6 py-8">
        <div className="rounded-md border border-[#cf222e] bg-[#ffebe9] px-4 py-3 text-sm text-[#cf222e]">
          {error}
        </div>
        <Link href="/10-actions-list" className="mt-4 inline-block text-sm text-[#0969da] hover:underline">
          ← Back to Actions
        </Link>
      </div>
    );
  }

  if (!run) {
    return (
      <div className="min-h-screen bg-[#f6f8fa] px-6 py-8 text-[#656d76]">
        Workflow run not found
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <header className="sticky top-0 z-50 h-16 flex items-center justify-between px-6 bg-white/85 backdrop-blur border-b border-[#d0d7de]">
        <div className="flex items-center gap-3 font-extrabold text-lg">
          <span>🐙</span>
          <strong>OpenHub</strong>
        </div>
        <Link
          href="/10-actions-list"
          className="px-3 py-1.5 text-sm rounded-md border border-[#d0d7de] hover:bg-[#f3f4f6]"
        >
          ← Actions
        </Link>
      </header>

      <div className="max-w-[1280px] mx-auto px-6 py-6">
        {error && (
          <div className="mb-4 rounded-md border border-[#cf222e] bg-[#ffebe9] px-4 py-3 text-sm text-[#cf222e]">
            {error}
          </div>
        )}

        <div className="bg-white border border-[#d0d7de] rounded-md overflow-hidden mb-6">
          <div className="px-5 py-4 border-b border-[#d0d7de] bg-[#f6f8fa] flex flex-wrap items-center justify-between gap-3">
            <div>
              <h1 className="text-xl font-semibold m-0">
                {run.name}{" "}
                <span className="text-[#656d76] font-normal">#{run.run_number}</span>
              </h1>
              <div className="mt-2 flex flex-wrap items-center gap-3 text-sm text-[#656d76]">
                <RunStatusBadge status={run.status} conclusion={run.conclusion} />
                <span title={run.event ?? "workflow"}>
                  {eventIcon(run.event)} {run.event ?? "workflow"}
                </span>
                <span className="bg-[#ddf4ff] text-[#0969da] px-1.5 py-0.5 rounded font-mono text-[11px]">
                  {run.head_branch}
                </span>
                <Link
                  href={`/07-repo-detail?sha=${run.head_sha}`}
                  className="font-mono text-[#0969da] hover:underline"
                >
                  {run.head_sha.slice(0, 7)}
                </Link>
                <span>{actorLogin(run)}</span>
                <span>{formatTimestamp(run.created_at)}</span>
                <span>{runDuration(run)}</span>
              </div>
            </div>
            <div className="flex gap-2">
              {canCancelRun(run) && (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={actionLoading}
                  onClick={handleCancel}
                >
                  Cancel
                </Button>
              )}
              {canRerunRun(run) && (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={actionLoading}
                  onClick={handleRerun}
                >
                  Re-run
                </Button>
              )}
            </div>
          </div>
        </div>

        <div className="space-y-3">
          <h2 className="text-base font-semibold m-0">Jobs</h2>
          {jobs.length === 0 ? (
            <div className="bg-white border border-[#d0d7de] rounded-md px-4 py-6 text-sm text-[#656d76]">
              No jobs found.
            </div>
          ) : (
            jobs.map((job) => (
              <Link
                key={job.id}
                href={`/${owner}/${repo}/actions/jobs/${job.id}`}
                className="block bg-white border border-[#d0d7de] rounded-md px-4 py-3 hover:bg-[#fafbfc] no-underline text-inherit"
              >
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div className="flex flex-wrap items-center gap-3">
                    <RunStatusBadge status={job.status} conclusion={job.conclusion} />
                    <span className="font-semibold text-sm">{job.name}</span>
                    <span className="text-xs text-[#656d76]">{jobRunnerLabel(job)}</span>
                  </div>
                  <span className="text-xs text-[#656d76]">
                    {formatDuration(job.started_at, job.completed_at)}
                  </span>
                </div>
              </Link>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
