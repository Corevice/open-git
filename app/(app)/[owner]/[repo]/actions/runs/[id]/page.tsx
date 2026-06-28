"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { RunStatusBadge } from "@/components/actions/RunStatusBadge";
import { useJobLogStream } from "@/hooks/useJobLogStream";
import { API_TOKEN_KEY } from "@/lib/api";

type WorkflowJob = {
  id: number;
  name: string;
  status: string;
  conclusion: string | null;
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
};

type WorkflowJobsResponse = {
  total_count: number;
  jobs: WorkflowJob[];
};

function EmptyState({ title, description }: { title: string; description: string }) {
  return (
    <div className="rounded-md border border-[#d0d7de] bg-white px-6 py-12 text-center">
      <h2 className="text-base font-semibold text-[#1f2328]">{title}</h2>
      <p className="mt-2 text-sm text-[#656d76]">{description}</p>
    </div>
  );
}

function StreamingIndicator() {
  return (
    <span className="inline-flex items-center gap-2 text-xs text-[#656d76]">
      <span
        className="inline-block h-2 w-2 animate-pulse rounded-full bg-[#0969da]"
        aria-hidden
      />
      <span>Streaming</span>
    </span>
  );
}

function LogViewer({ lines, streaming }: { lines: string[]; streaming: boolean }) {
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = scrollRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }, [lines]);

  return (
    <div
      ref={scrollRef}
      className="max-h-[480px] overflow-y-auto rounded-md border border-[#d0d7de] bg-[#0d1117] p-4 font-mono text-sm text-[#e6edf3]"
    >
      {lines.length === 0 ? (
        <div className="text-[#656d76]">
          {streaming ? "Waiting for log output…" : "No log output."}
        </div>
      ) : (
        lines.map((line, index) => (
          <div key={index} className="whitespace-pre-wrap break-all">
            {line}
          </div>
        ))
      )}
    </div>
  );
}

export default function ActionRunDetailPage() {
  const params = useParams();
  const owner = params.owner as string;
  const repo = params.repo as string;
  const runId = params.id as string;

  const [run, setRun] = useState<WorkflowRunDetail | null>(null);
  const [jobs, setJobs] = useState<WorkflowJob[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedJobIds, setExpandedJobIds] = useState<Set<number>>(new Set());
  const [selectedJobId, setSelectedJobId] = useState<number | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(null);

      const headers: Record<string, string> = {
        Accept: "application/vnd.github+json",
      };
      const token =
        typeof window !== "undefined" ? localStorage.getItem(API_TOKEN_KEY) : null;
      if (token) {
        headers.Authorization = `Bearer ${token}`;
      }

      try {
        const [runRes, jobsRes] = await Promise.all([
          fetch(`/api/v3/repos/${owner}/${repo}/actions/runs/${runId}`, {
            headers,
          }),
          fetch(`/api/v3/repos/${owner}/${repo}/actions/runs/${runId}/jobs`, {
            headers,
          }),
        ]);

        if (!runRes.ok) throw new Error("Failed to load workflow run");
        if (!jobsRes.ok) throw new Error("Failed to load workflow jobs");

        const runData = (await runRes.json()) as WorkflowRunDetail;
        const jobsData = (await jobsRes.json()) as WorkflowJobsResponse;

        if (cancelled) return;

        setRun(runData);
        const loadedJobs = jobsData.jobs ?? [];
        setJobs(loadedJobs);

        const inProgressJob = loadedJobs.find(
          (job) => job.status === "in_progress" || job.status === "queued",
        );
        const initialJob = inProgressJob ?? loadedJobs[0] ?? null;
        if (initialJob) {
          setSelectedJobId(initialJob.id);
          setExpandedJobIds(new Set([initialJob.id]));
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

  const activeJob =
    jobs.find((job) => job.id === selectedJobId) ??
    jobs.find((job) => job.status === "in_progress" || job.status === "queued") ??
    jobs[0] ??
    null;

  const streamEnabled =
    activeJob != null &&
    (activeJob.status === "in_progress" ||
      activeJob.status === "queued" ||
      activeJob.id === selectedJobId);

  const { lines, status } = useJobLogStream({
    owner,
    repo,
    jobId: activeJob?.id ?? null,
    enabled: streamEnabled,
  });

  function toggleJobExpanded(jobId: number) {
    setExpandedJobIds((prev) => {
      const next = new Set(prev);
      if (next.has(jobId)) {
        next.delete(jobId);
      } else {
        next.add(jobId);
      }
      return next;
    });
  }

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

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <div className="mx-auto max-w-[1280px] px-6 py-6">
        <div className="mb-4">
          <Link
            href={`/${owner}/${repo}/actions`}
            className="text-sm text-[#0969da] hover:underline"
          >
            ← Actions
          </Link>
          <h1 className="mt-2 text-2xl font-semibold">
            {run.name}{" "}
            <span className="font-normal text-[#656d76]">#{run.run_number}</span>
          </h1>
          <p className="mt-1 text-sm text-[#656d76]">
            <span className="rounded bg-[#ddf4ff] px-1.5 py-0.5 font-mono text-[11px] text-[#0969da]">
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

        {jobs.length === 0 ? (
          <EmptyState
            title="No jobs found"
            description="This workflow run does not have any jobs yet."
          />
        ) : (
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-[320px_1fr]">
            <aside className="space-y-2">
              {jobs.map((job) => {
                const isExpanded = expandedJobIds.has(job.id);
                const isSelected = selectedJobId === job.id;

                return (
                  <section
                    key={job.id}
                    className="overflow-hidden rounded-md border border-[#d0d7de] bg-white"
                  >
                    <button
                      type="button"
                      onClick={() => {
                        setSelectedJobId(job.id);
                        toggleJobExpanded(job.id);
                      }}
                      className={`flex w-full items-center justify-between gap-3 px-4 py-3 text-left text-sm hover:bg-[#fafbfc] ${
                        isSelected ? "bg-[#ddf4ff]" : ""
                      }`}
                      aria-expanded={isExpanded}
                    >
                      <span className="font-medium">{job.name}</span>
                      <RunStatusBadge status={job.status} conclusion={job.conclusion} />
                    </button>
                    {isExpanded && (
                      <div className="border-t border-[#d8dee4] bg-[#f6f8fa] px-4 py-3 text-xs text-[#656d76]">
                        Job #{job.id}
                        {isSelected && activeJob?.id === job.id && status === "streaming" && (
                          <span className="ml-3 inline-flex items-center gap-1">
                            <span className="inline-block h-1.5 w-1.5 animate-pulse rounded-full bg-[#0969da]" />
                            Live
                          </span>
                        )}
                      </div>
                    )}
                  </section>
                );
              })}
            </aside>

            <main>
              <div className="mb-3 flex items-center justify-between">
                <div className="text-sm font-semibold text-[#656d76]">
                  {activeJob ? `Logs — ${activeJob.name}` : "Logs"}
                </div>
                {status === "streaming" && <StreamingIndicator />}
              </div>
              {activeJob ? (
                <LogViewer lines={lines} streaming={status === "streaming"} />
              ) : null}
            </main>
          </div>
        )}
      </div>
    </div>
  );
}
