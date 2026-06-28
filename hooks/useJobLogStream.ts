"use client";

import { useEffect, useRef, useState } from "react";

export type JobLogStreamStatus = "connecting" | "streaming" | "done" | "error";

type UseJobLogStreamParams = {
  owner: string;
  repo: string;
  jobId: string | number | null;
  enabled: boolean;
};

export function useJobLogStream({
  owner,
  repo,
  jobId,
  enabled,
}: UseJobLogStreamParams) {
  const [lines, setLines] = useState<string[]>([]);
  const [status, setStatus] = useState<JobLogStreamStatus>("connecting");
  const sourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    if (!enabled || jobId == null) {
      sourceRef.current?.close();
      sourceRef.current = null;
      return;
    }

    setLines([]);
    setStatus("connecting");

    const url = `/api/v3/repos/${owner}/${repo}/actions/jobs/${jobId}/logs`;
    const source = new EventSource(url);
    sourceRef.current = source;

    source.onopen = () => {
      setStatus("streaming");
    };

    source.onmessage = (event) => {
      setLines((prev) => [...prev, event.data]);
      setStatus("streaming");
    };

    source.addEventListener("concluded", () => {
      setStatus("done");
      source.close();
      sourceRef.current = null;
    });

    source.onerror = () => {
      setStatus("done");
      source.close();
      sourceRef.current = null;
    };

    return () => {
      source.close();
      sourceRef.current = null;
    };
  }, [owner, repo, jobId, enabled]);

  return { lines, status };
}
