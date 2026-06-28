"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import LogStreamStatus from "./LogStreamStatus";

type LogLine = {
  step: number;
  line: number;
  ts: string;
  stream: "stdout" | "stderr";
  text: string;
};

type JobLogViewerProps = {
  runId: string;
  jobId: string;
  repoOwner: string;
  repoName: string;
};

type StreamStatus = "streaming" | "completed" | "reconnecting" | "failed";

const MAX_BACKOFF_MS = 30_000;

export function stripAnsi(text: string): string {
  return text.replace(/\x1b\[[0-9;?]*[ -/]*[@-~]/g, "");
}

export default function JobLogViewer({
  runId,
  jobId,
  repoOwner,
  repoName,
}: JobLogViewerProps) {
  const [lines, setLines] = useState<LogLine[]>([]);
  const [status, setStatus] = useState<StreamStatus>("streaming");
  const lastLineRef = useRef(0);
  const scrollRef = useRef<HTMLDivElement>(null);
  const reconnectAttemptRef = useRef(0);
  const completedRef = useRef(false);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const connect = useCallback(() => {
    const baseUrl = `/api/v1/repos/${repoOwner}/${repoName}/actions/runs/${runId}/jobs/${jobId}/logs/stream`;
    const url =
      lastLineRef.current > 0
        ? `${baseUrl}?from_line=${lastLineRef.current}`
        : baseUrl;

    const source = new EventSource(url);

    const handleLog = (event: MessageEvent) => {
      const parsed = JSON.parse(event.data) as LogLine;
      lastLineRef.current = Math.max(lastLineRef.current, parsed.line);
      setLines((prev) => {
        if (prev.some((entry) => entry.line === parsed.line)) {
          return prev;
        }
        return [...prev, parsed].sort((a, b) => a.line - b.line);
      });
      reconnectAttemptRef.current = 0;
    };

    const handleDone = () => {
      completedRef.current = true;
      setStatus("completed");
      source.close();
    };

    source.addEventListener("log", handleLog as EventListener);
    source.addEventListener("done", handleDone);

    source.onerror = () => {
      if (completedRef.current) {
        return;
      }
      source.close();
      setStatus("reconnecting");
      const delay = Math.min(
        1000 * 2 ** reconnectAttemptRef.current,
        MAX_BACKOFF_MS,
      );
      reconnectAttemptRef.current += 1;
      reconnectTimerRef.current = setTimeout(() => {
        setStatus("streaming");
        connect();
      }, delay);
    };

    return source;
  }, [runId, jobId, repoOwner, repoName]);

  useEffect(() => {
    completedRef.current = false;
    reconnectAttemptRef.current = 0;
    lastLineRef.current = 0;
    setLines([]);
    setStatus("streaming");

    const source = connect();

    return () => {
      if (reconnectTimerRef.current !== null) {
        clearTimeout(reconnectTimerRef.current);
      }
      source.close();
    };
  }, [connect]);

  useEffect(() => {
    const el = scrollRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }, [lines]);

  const groupedLines = useMemo(() => {
    const groups = new Map<number, LogLine[]>();
    for (const entry of lines) {
      const group = groups.get(entry.step) ?? [];
      group.push(entry);
      groups.set(entry.step, group);
    }
    return Array.from(groups.entries()).sort(([a], [b]) => a - b);
  }, [lines]);

  return (
    <div className="relative">
      <div className="mb-2">
        <LogStreamStatus status={status} />
      </div>
      <div
        ref={scrollRef}
        className="max-h-[480px] overflow-y-auto rounded-md border border-[#d0d7de] bg-[#0d1117] p-4 font-mono text-sm text-[#e6edf3]"
      >
        {lines.length === 0 ? (
          <div className="text-[#656d76]">Waiting for log output…</div>
        ) : (
          groupedLines.map(([step, stepLines]) => (
            <details key={step} open className="mb-2">
              <summary className="cursor-pointer select-none text-[#8b949e]">
                Step {step}
              </summary>
              {stepLines.map((entry) => (
                <div key={entry.line} className="flex gap-3 whitespace-pre-wrap break-all">
                  <span className="inline-block w-12 shrink-0 select-none text-right font-mono text-[#656d76]">
                    {entry.line}
                  </span>
                  <span
                    className={
                      entry.stream === "stderr" ? "text-[#f85149]" : undefined
                    }
                  >
                    {stripAnsi(entry.text)}
                  </span>
                </div>
              ))}
            </details>
          ))
        )}
      </div>
    </div>
  );
}
