"use client";

import { useEffect, useRef } from "react";

import { useJobLogStream } from "@/hooks/useJobLogStream";

export function stripAnsi(text: string): string {
  return text.replace(/\x1b\[[0-9;]*m/g, "");
}

type JobLogViewerProps = {
  owner: string;
  repo: string;
  jobId: string | null;
  isActive: boolean;
};

const MAX_VISIBLE_LINES = 2000;

export function JobLogViewer({ owner, repo, jobId, isActive }: JobLogViewerProps) {
  const { lines, streaming } = useJobLogStream({ owner, repo, jobId, isActive });
  const scrollRef = useRef<HTMLDivElement>(null);

  const visibleLineCount = Math.min(lines.length, MAX_VISIBLE_LINES);
  const visibleLines = lines.slice(-visibleLineCount);

  useEffect(() => {
    const el = scrollRef.current;
    if (el && el.scrollHeight - el.scrollTop - el.clientHeight < 100) {
      el.scrollTop = el.scrollHeight;
    }
  }, [lines]);

  return (
    <div className="relative">
      {streaming && (
        <div className="absolute right-3 top-3 flex items-center gap-2 text-xs text-[#656d76]">
          <span
            className="inline-block h-4 w-4 animate-spin rounded-full border-2 border-[#d0d7de] border-t-[#0969da]"
            aria-hidden
          />
          <span>Streaming logs…</span>
        </div>
      )}
      <div
        ref={scrollRef}
        className="max-h-[480px] overflow-y-auto rounded-md border border-[#d0d7de] bg-[#0d1117] p-4 font-mono text-sm text-[#e6edf3]"
      >
        {visibleLines.length === 0 ? (
          <div className="text-[#656d76]">
            {streaming ? "Waiting for log output…" : "No log output."}
          </div>
        ) : (
          visibleLines.map((line, index) => (
            <div key={index} className="whitespace-pre-wrap break-all">
              {stripAnsi(line)}
            </div>
          ))
        )}
      </div>
    </div>
  );
}
