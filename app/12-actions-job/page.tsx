"use client";

import { Suspense, useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { RunStatusBadge } from "@/components/actions/RunStatusBadge";
import { API_TOKEN_KEY } from "@/lib/api";
import { env } from "@/lib/env";

type WorkflowStep = {
  number: number;
  name: string;
  status: string;
  conclusion: string | null;
};

type JobDetail = {
  id: number;
  name: string;
  status: string;
  conclusion: string | null;
  started_at?: string | null;
  completed_at?: string | null;
  steps?: WorkflowStep[];
};

function StatusBadge({
  status,
  conclusion,
}: {
  status: string;
  conclusion: string | null;
}) {
  return <RunStatusBadge status={status} conclusion={conclusion} />;
}

function formatTimestamp(value: string | null | undefined): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function buildAuthHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    Accept: "application/vnd.github+json",
  };
  const token =
    typeof window !== "undefined" ? localStorage.getItem(API_TOKEN_KEY) : null;
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return headers;
}

export function JobLogsPageContent() {
  const searchParams = useSearchParams();
  const jobId = searchParams.get("jobId") ?? "1";
  const owner = searchParams.get("owner") ?? "octocat";
  const repo = searchParams.get("repo") ?? "hello-world";

  const [job, setJob] = useState<JobDetail | null>(null);
  const [steps, setSteps] = useState<WorkflowStep[]>([]);
  const [logLines, setLogLines] = useState<string[]>([]);
  const [initialOffset, setInitialOffset] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [autoScroll, setAutoScroll] = useState(true);
  const [streamReady, setStreamReady] = useState(false);

  const preRef = useRef<HTMLPreElement>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function loadInitialData() {
      setLoading(true);
      setError(null);
      setStreamReady(false);

      const headers = buildAuthHeaders();
      const jobUrl = `/api/v3/repos/${owner}/${repo}/actions/jobs/${jobId}`;
      const logsUrl = `/api/v3/repos/${owner}/${repo}/actions/jobs/${jobId}/logs`;

      try {
        const [jobRes, logsRes] = await Promise.all([
          fetch(jobUrl, { headers }),
          fetch(logsUrl, { headers }),
        ]);

        if (!jobRes.ok) {
          throw new Error("Failed to load job details");
        }
        if (!logsRes.ok) {
          throw new Error("Failed to load job logs");
        }

        const jobData = (await jobRes.json()) as JobDetail;
        const logsText = await logsRes.text();

        if (cancelled) return;

        setJob(jobData);
        setSteps(jobData.steps ?? []);

        const lines = logsText.length > 0 ? logsText.split("\n") : [];
        if (logsText.endsWith("\n") && lines.length > 0) {
          lines.pop();
        }
        setLogLines(lines);
        setInitialOffset(logsText.length);
        setStreamReady(true);
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : "Failed to load job");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    loadInitialData();

    return () => {
      cancelled = true;
    };
  }, [owner, repo, jobId]);

  useEffect(() => {
    if (!streamReady) {
      return;
    }

    const apiBase = env.NEXT_PUBLIC_API_BASE_URL.replace(/\/$/, "");
    const streamUrl = `${apiBase}/repos/${owner}/${repo}/actions/jobs/${jobId}/logs/stream?offset=${initialOffset}`;
    const source = new EventSource(streamUrl);
    eventSourceRef.current = source;

    source.onmessage = (event) => {
      setLogLines((prev) => [...prev, event.data]);
    };

    source.addEventListener("done", () => {
      source.close();
      eventSourceRef.current = null;
    });

    return () => {
      source.close();
      eventSourceRef.current = null;
    };
  }, [owner, repo, jobId, initialOffset, streamReady]);

  useEffect(() => {
    const el = preRef.current;
    if (!el || !autoScroll) {
      return;
    }

    const atBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - 50;
    if (atBottom) {
      el.scrollTop = el.scrollHeight;
    }
  }, [logLines, autoScroll]);

  const handleScroll = useCallback(() => {
    const el = preRef.current;
    if (!el) {
      return;
    }

    const atBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - 50;
    if (!atBottom) {
      setAutoScroll(false);
    }
  }, []);

  if (loading) {
    return (
      <div className="min-h-screen bg-[#f6f8fa] px-6 py-8 text-[#656d76]">
        Loading job logs…
      </div>
    );
  }

  if (error || !job) {
    return (
      <div className="min-h-screen bg-[#f6f8fa] px-6 py-8">
        <div className="rounded-md border border-[#cf222e] bg-[#ffebe9] px-4 py-3 text-sm text-[#cf222e]">
          {error ?? "Job not found"}
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <div className="mx-auto max-w-[1280px] px-6 py-6">
        <header className="mb-6 flex flex-wrap items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold">{job.name}</h1>
            <div className="mt-2 flex flex-wrap items-center gap-3 text-sm text-[#656d76]">
              <StatusBadge status={job.status} conclusion={job.conclusion} />
              <span>Started: {formatTimestamp(job.started_at)}</span>
              <span>Completed: {formatTimestamp(job.completed_at)}</span>
            </div>
          </div>
          <button
            type="button"
            onClick={() => setAutoScroll((current) => !current)}
            className={`rounded-full px-3 py-1 text-xs font-semibold text-white ${
              autoScroll ? "bg-green-600" : "bg-gray-500"
            }`}
            aria-pressed={autoScroll}
          >
            {autoScroll ? "Live" : "Paused"}
          </button>
        </header>

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-[280px_1fr]">
          <aside className="space-y-2">
            <h2 className="text-xs uppercase tracking-wide text-[#656d76]">Steps</h2>
            {steps.length === 0 ? (
              <p className="text-sm text-[#656d76]">No steps available.</p>
            ) : (
              steps.map((step) => (
                <div
                  key={step.number}
                  className="flex items-center justify-between gap-3 rounded-md border border-[#d0d7de] bg-white px-3 py-2 text-sm"
                >
                  <span>{step.name}</span>
                  <StatusBadge status={step.status} conclusion={step.conclusion} />
                </div>
              ))
            )}
          </aside>

          <main>
            <pre
              ref={preRef}
              onScroll={handleScroll}
              className="font-mono text-sm bg-gray-900 text-gray-100 p-4 overflow-auto h-[60vh]"
            >
              {logLines.length === 0
                ? "No log output."
                : logLines.map((line, index) => (
                    <span key={index} className="block whitespace-pre-wrap break-all">
                      {line}
                    </span>
                  ))}
            </pre>
          </main>
        </div>
      </div>
    </div>
  );
}

export default function Page() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen bg-[#f6f8fa] px-6 py-8 text-[#656d76]">
          Loading job logs…
        </div>
      }
    >
      <JobLogsPageContent />
    </Suspense>
  );
}
