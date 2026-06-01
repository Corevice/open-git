"use client";

import { useEffect, useRef, useState } from "react";

type LogViewerProps = {
  runId: string;
  repoOwner: string;
  repoName: string;
};

export default function LogViewer({ runId, repoOwner, repoName }: LogViewerProps) {
  const [lines, setLines] = useState<string[]>([]);
  const [streaming, setStreaming] = useState(true);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const url = `/api/repos/${repoOwner}/${repoName}/actions/runs/${runId}/logs?stream=true`;
    const source = new EventSource(url);

    source.onmessage = (event) => {
      setLines((prev) => [...prev, event.data]);
    };

    source.onerror = () => {
      setStreaming(false);
      source.close();
    };

    return () => {
      source.close();
    };
  }, [runId, repoOwner, repoName]);

  useEffect(() => {
    const el = scrollRef.current;
    if (el) {
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
    </div>
  );
}
