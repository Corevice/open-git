"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";
import { cancelRun, rerunFailedJobs, rerunRun } from "@/lib/api/actions";

type RunActionsProps = {
  owner: string;
  repo: string;
  runId: string;
  status: string;
  conclusion: string | null;
  userCanWrite: boolean;
  onActionComplete: () => void;
};

export function RunActions({
  owner,
  repo,
  runId,
  status,
  conclusion,
  userCanWrite,
  onActionComplete,
}: RunActionsProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const showRerunAll = status === "completed";
  const showRerunFailed = status === "completed" && conclusion === "failure";
  const showCancel = status === "queued" || status === "in_progress";
  const disabled = !userCanWrite || loading;

  const runAction = async (action: () => Promise<void>) => {
    setLoading(true);
    setError(null);
    try {
      await action();
      onActionComplete();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Action failed");
    } finally {
      setLoading(false);
    }
  };

  if (!showRerunAll && !showCancel) {
    return null;
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      {showRerunAll && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={disabled}
          onClick={() => runAction(() => rerunRun(owner, repo, runId))}
        >
          Re-run all jobs
        </Button>
      )}
      {showRerunFailed && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={disabled}
          onClick={() => runAction(() => rerunFailedJobs(owner, repo, runId))}
        >
          Re-run failed jobs
        </Button>
      )}
      {showCancel && (
        <Button
          type="button"
          variant="destructive"
          size="sm"
          disabled={disabled}
          onClick={() => runAction(() => cancelRun(owner, repo, runId))}
        >
          Cancel
        </Button>
      )}
      {error && (
        <p className="w-full text-sm text-[#cf222e]" role="alert">
          {error}
        </p>
      )}
    </div>
  );
}
