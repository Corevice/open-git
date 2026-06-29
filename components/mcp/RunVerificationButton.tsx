"use client";

import { FormEvent, useEffect, useRef, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiClient, isApiError } from "@/lib/api-client";

interface RunVerificationButtonProps {
  onComplete?: () => void;
}

export function RunVerificationButton({ onComplete }: RunVerificationButtonProps) {
  const [repository, setRepository] = useState("");
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, []);

  const clearPolling = () => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (isRunning || !repository.trim()) {
      return;
    }

    setError(null);
    setIsRunning(true);

    try {
      const { job_id: jobId } = await apiClient.runMCPVerification({
        repository: repository.trim(),
        targets: ["graphql", "rest", "auth"],
      });

      intervalRef.current = setInterval(async () => {
        try {
          const status = await apiClient.getMCPJobStatus(jobId);
          if (status.status === "completed" || status.status === "errored") {
            clearPolling();
            setIsRunning(false);
            onComplete?.();
          }
        } catch (pollError) {
          clearPolling();
          setIsRunning(false);
          if (pollError instanceof Error) {
            setError(pollError.message);
          } else if (isApiError(pollError)) {
            setError(pollError.message);
          } else {
            setError("Failed to poll verification job status.");
          }
        }
      }, 2000);
    } catch (submitError) {
      setIsRunning(false);
      if (submitError instanceof Error) {
        setError(submitError.message);
      } else if (isApiError(submitError)) {
        setError(submitError.message);
      } else {
        setError("Failed to start verification.");
      }
    }
  };

  return (
    <div className="space-y-4">
      <form onSubmit={handleSubmit} className="flex flex-wrap items-end gap-3">
        <div className="min-w-[280px] flex-1">
          <Input
            type="text"
            value={repository}
            onChange={(event) => setRepository(event.target.value)}
            placeholder="owner/repo"
            disabled={isRunning}
            required
          />
        </div>
        <Button type="submit" disabled={isRunning}>
          {isRunning && (
            <div className="animate-spin h-4 w-4 rounded-full border-2 border-primary border-t-transparent" />
          )}
          Run verification
        </Button>
      </form>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
    </div>
  );
}
