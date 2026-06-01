"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import LogViewer from "@/components/actions/LogViewer";

type WorkflowStep = {
  name: string;
  status: string;
  conclusion: string | null;
  number: number;
};

type WorkflowJob = {
  id: number;
  name: string;
  status: string;
  conclusion: string | null;
  steps?: WorkflowStep[];
};

type WorkflowRunDetail = {
  id: number;
  name: string;
  run_number: number;
  status: string;
  conclusion: string | null;
  head_branch: string;
  head_sha: string;
  parse_error_line?: number;
  parse_error_message?: string;
  jobs?: WorkflowJob[];
};

function stepStatusIcon(status: string, conclusion: string | null): string {
  if (status === "in_progress" || status === "queued") return "🟡";
  if (conclusion === "success") return "✅";
  if (conclusion === "failure") return "❌";
  if (conclusion === "skipped" || conclusion === "cancelled") return "⊘";
  return "○";
}

function jobStatusIcon(job: WorkflowJob): string {
  return stepStatusIcon(job.status, job.conclusion);
}

export default function ActionRunDetailPage() {
  const params = useParams();
  const owner = params.owner as string;
  const repo = params.repo as string;
  const runId = params.id as string;

  const [run, setRun] = useState<WorkflowRunDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedJobId, setSelectedJobId] = useState<number | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(null);
      try {
        const res = await fetch(`/api/repos/${owner}/${repo}/actions/runs/${runId}`);
        if (!res.ok) throw new Error("Failed to load workflow run");
        const data = (await res.json()) as WorkflowRunDetail;
        if (cancelled) return;
        setRun(data);
        if (data.jobs?.length) {
          setSelectedJobId(data.jobs[0].id);
        }
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : "Failed to load workflow run");
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    load();
    return () => {
      cancelled = true;
    };
  }, [owner, repo, runId]);

  if (loading) {
    return (
      <div className="min-h-screen bg-[#f6f8fa] px-6 py-8 text-[#656d76]">
        Loading workflow run…
      </div>
    );
  }

  if (error || !run) {
    return (
      <div className="min-h-screen bg-[#f6f8fa] px-6 py-8">
        <div className="rounded-md border border-[#cf222e] bg-[#ffebe9] px-4 py-3 text-sm text-[#cf222e]">
          {error ?? "Workflow run not found"}
        </div>
        <Link
          href={`/${owner}/${repo}/actions`}
          className="mt-4 inline-block text-sm text-[#0969da] hover:underline"
        >
          ← Back to Actions
        </Link>
      </div>
    );
  }

  const selectedJob = run.jobs?.find((j) => j.id === selectedJobId) ?? run.jobs?.[0];

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <div className="max-w-[1280px] mx-auto px-6 py-6">
        <div className="mb-4">
          <Link
            href={`/${owner}/${repo}/actions`}
            className="text-sm text-[#0969da] hover:underline"
          >
            ← Actions
          </Link>
          <h1 className="text-2xl font-semibold mt-2">
            {run.name}{" "}
            <span className="text-[#656d76] font-normal">#{run.run_number}</span>
          </h1>
          <p className="text-sm text-[#656d76] mt-1">
            <span className="bg-[#ddf4ff] text-[#0969da] px-1.5 py-0.5 rounded font-mono text-[11px]">
              {run.head_branch}
            </span>
            {" · "}
            <span className="font-mono">{run.head_sha.slice(0, 7)}</span>
          </p>
        </div>

        {run.conclusion === "workflow_error" && (
          <div
            role="alert"
            className="mb-4 rounded-md border border-[#cf222e] bg-[#ffebe9] px-4 py-3 text-sm text-[#cf222e]"
          >
            Workflow parse error on line {run.parse_error_line ?? "?"}:{" "}
            {run.parse_error_message ?? "Unknown parse error"}
          </div>
        )}

        <div className="grid grid-cols-[280px_1fr] gap-6">
          <aside className="bg-white border border-[#d0d7de] rounded-md overflow-hidden">
            <div className="px-4 py-3 bg-[#f6f8fa] border-b border-[#d0d7de] text-sm font-semibold">
              Jobs
            </div>
            {run.jobs && run.jobs.length > 0 ? (
              <ul>
                {run.jobs.map((job) => (
                  <li key={job.id} className="border-b border-[#d8dee4] last:border-b-0">
                    <button
                      type="button"
                      onClick={() => setSelectedJobId(job.id)}
                      className={`w-full text-left px-4 py-3 text-sm hover:bg-[#fafbfc] ${
                        selectedJob?.id === job.id ? "bg-[#ddf4ff] font-semibold" : ""
                      }`}
                    >
                      <span className="mr-2">{jobStatusIcon(job)}</span>
                      {job.name}
                    </button>
                    {selectedJob?.id === job.id && job.steps && job.steps.length > 0 && (
                      <ul className="bg-[#f6f8fa] border-t border-[#d8dee4]">
                        {job.steps.map((step) => (
                          <li
                            key={step.number}
                            className="flex items-center gap-2 px-6 py-2 text-xs text-[#656d76]"
                          >
                            <span>{stepStatusIcon(step.status, step.conclusion)}</span>
                            <span>{step.name}</span>
                          </li>
                        ))}
                      </ul>
                    )}
                  </li>
                ))}
              </ul>
            ) : (
              <div className="px-4 py-6 text-sm text-[#656d76]">No jobs</div>
            )}
          </aside>

          <main>
            <div className="mb-3 text-sm font-semibold text-[#656d76]">
              {selectedJob ? `Logs — ${selectedJob.name}` : "Logs"}
            </div>
            <LogViewer runId={runId} repoOwner={owner} repoName={repo} />
          </main>
        </div>
      </div>
    </div>
  );
}
