import { ApiError } from "@/lib/api";
import { env } from "@/lib/env";
import type { WorkflowJobResponse } from "@/types/runner";

async function getWorkflowJob(
  org: string,
  jobId: string,
): Promise<WorkflowJobResponse | null> {
  const headers: Record<string, string> = {
    Accept: "application/json",
  };

  const baseUrl = env.NEXT_PUBLIC_API_BASE_URL.replace(/\/$/, "");
  const response = await fetch(
    `${baseUrl}/api/v1/${org}/actions/jobs/${jobId}`,
    { headers, cache: "no-store" },
  );

  if (response.status === 404) {
    return null;
  }

  if (!response.ok) {
    let message = response.statusText;
    try {
      const errorBody = (await response.json()) as { message?: string };
      message = errorBody.message ?? message;
    } catch {
      // ignore JSON parse errors
    }
    throw new ApiError(response.status, message);
  }

  return response.json() as Promise<WorkflowJobResponse>;
}

export default async function RunDetailPage({
  params,
  searchParams,
}: {
  params: Promise<{ org: string; repo: string; runId: string }>;
  searchParams: Promise<{ jobId?: string }>;
}) {
  const { org, repo, runId } = await params;
  const { jobId } = await searchParams;

  // TODO: Replace stub with full workflow run detail UI (jobs, logs, steps).

  let job: WorkflowJobResponse | null = null;
  let jobError: string | null = null;

  if (jobId) {
    try {
      job = await getWorkflowJob(org, jobId);
    } catch (err) {
      jobError =
        err instanceof Error ? err.message : "Failed to load workflow job.";
    }
  }

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <div className="mx-auto grid max-w-[1280px] grid-cols-[280px_1fr] gap-6 px-6 py-6">
        <aside className="overflow-hidden rounded-md border border-[#d0d7de] bg-white">
          <div className="border-b border-[#d0d7de] bg-[#f6f8fa] px-4 py-3 text-sm font-semibold">
            Runner
          </div>
          <div className="space-y-3 px-4 py-4 text-sm">
            {jobError ? (
              <p className="text-[#cf222e]">{jobError}</p>
            ) : job ? (
              <>
                <div>
                  <div className="text-xs uppercase text-[#656d76]">
                    Assigned runner
                  </div>
                  <div className="mt-1 font-mono text-xs">
                    {job.assigned_runner_id ?? "—"}
                  </div>
                </div>
                <div>
                  <div className="text-xs uppercase text-[#656d76]">
                    Runner type
                  </div>
                  <div className="mt-1">{job.runner_type ?? "—"}</div>
                </div>
              </>
            ) : (
              <p className="text-[#656d76]">
                Provide a <span className="font-mono">jobId</span> query
                parameter to load runner assignment details.
              </p>
            )}
          </div>
        </aside>

        <main>
          <h1 className="text-2xl font-semibold">Run detail (coming soon)</h1>
          <p className="mt-2 text-sm text-[#656d76]">
            Workflow run {runId} for {org}/{repo}.
          </p>
        </main>
      </div>
    </div>
  );
}
