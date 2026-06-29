"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { ArtifactList } from "@/components/actions/ArtifactList";
import { JobLogViewer } from "@/components/actions/JobLogViewer";
import { RunActions } from "@/components/actions/RunActions";
import { RunStatusBadge } from "@/components/actions/RunStatusBadge";
import {
  getRun,
  listArtifacts,
  listJobs,
  type Artifact,
  type WorkflowJob,
  type WorkflowRun,
} from "@/lib/api/actions";

function getUserCanWrite(): boolean {
  if (typeof window === "undefined") return true;
  const stored = localStorage.getItem("open-git-user-can-write");
  return stored === null ? true : stored === "true";
}

function EmptyState({ title, description }: { title: string; description: string }) {
  return (
    <div className="rounded-md border border-[#d0d7de] bg-white px-6 py-12 text-center">
      <h2 className="text-base font-semibold text-[#1f2328]">{title}</h2>
      <p className="mt-2 text-sm text-[#656d76]">{description}</p>
    </div>
  );
}

export default function ActionRunDetailPage() {
  const params = useParams();
  const owner = params.owner as string;
  const repo = params.repo as string;
  const runId = params.id as string;

  const [run, setRun] = useState<WorkflowRun | null>(null);
  const [jobs, setJobs] = useState<WorkflowJob[]>([]);
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);
  const [loading, setLoading] = useState(true);
  const [artifactsLoading, setArtifactsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expandedJobIds, setExpandedJobIds] = useState<Set<number>>(new Set());
  const [selectedJobId, setSelectedJobId] = useState<number | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const [runData, jobsData] = await Promise.all([
        getRun(owner, repo, runId),
        listJobs(owner, repo, runId),
      ]);

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
      setError(e instanceof Error ? e.message : "Failed to load workflow run");
    } finally {
      setLoading(false);
    }
  }, [owner, repo, runId]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    if (!run) return;

    let cancelled = false;

    async function loadArtifacts() {
      setArtifactsLoading(true);
      try {
        const data = await listArtifacts(owner, repo, runId);
        if (!cancelled) {
          setArtifacts(data.artifacts ?? []);
        }
      } catch {
        if (!cancelled) {
          setArtifacts([]);
        }
      } finally {
        if (!cancelled) {
          setArtifactsLoading(false);
        }
      }
    }

    loadArtifacts();

    return () => {
      cancelled = true;
    };
  }, [owner, repo, runId, run]);

  const selectedJob = jobs.find((job) => job.id === selectedJobId) ?? null;

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

  const parseErrorRun = run as WorkflowRun & {
    parse_error_line?: number;
    parse_error_message?: string;
  };

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
            Workflow parse error on line {parseErrorRun.parse_error_line ?? "?"}:{" "}
            {parseErrorRun.parse_error_message ?? "Unknown parse error"}
          </div>
        )}

        <RunActions
          owner={owner}
          repo={repo}
          runId={runId}
          status={run.status}
          conclusion={run.conclusion}
          userCanWrite={getUserCanWrite()}
          onActionComplete={load}
        />

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
                      </div>
                    )}
                  </section>
                );
              })}
            </aside>

            <main>
              <div className="mb-3 text-sm font-semibold text-[#656d76]">
                {selectedJob ? `Logs — ${selectedJob.name}` : "Logs"}
              </div>
              {selectedJobId != null && (
                <JobLogViewer
                  owner={owner}
                  repo={repo}
                  jobId={String(selectedJobId)}
                  isActive={selectedJob?.status === "in_progress"}
                />
              )}
            </main>
          </div>
        )}

        <div className="mt-6">
          <ArtifactList
            owner={owner}
            repo={repo}
            artifacts={artifacts}
            loading={artifactsLoading}
          />
        </div>
      </div>
    </div>
  );
}
