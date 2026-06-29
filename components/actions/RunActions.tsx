"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";
import { cancelRun, rerunRun } from "@/lib/api/actions";

type RunActionsProps = {
  owner: string;
  repo: string;
  runId: string;
  status: string;
  conclusion: string | null;
  userCanWrite: boolean;
  onActionComplete: () => void | Promise<void>;
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
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isActive = status === "in_progress" || status === "queued";
  const isCompleted = status === "completed";
  const canRerun = isCompleted && conclusion !== "success";

  if (!userCanWrite) {
    return null;
  }

  async function run(action: () => Promise<void>) {
    setPending(true);
    setError(null);
    try {
      await action();
      await onActionComplete();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Action failed");
    } finally {
      setPending(false);
    }
  }

  if (!isActive && !canRerun) {
    return null;
  }

  return (
    <div className="mb-4 flex flex-wrap items-center gap-3">
      {isActive ? (
        <Button
          variant="destructive"
          size="sm"
          disabled={pending}
          onClick={() => run(() => cancelRun(owner, repo, runId))}
        >
          Cancel run
        </Button>
      ) : null}

      {canRerun ? (
        <Button
          variant="outline"
          size="sm"
          disabled={pending}
          onClick={() => run(() => rerunRun(owner, repo, runId))}
        >
          Re-run jobs
        </Button>
      ) : null}

      {error ? <span className="text-sm text-[#cf222e]">{error}</span> : null}
    </div>
  );
}
