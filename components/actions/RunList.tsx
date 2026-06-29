"use client";

import { RunStatusBadge } from "@/components/actions/RunStatusBadge";
import { Button } from "@/components/ui/button";

export type WorkflowRun = {
  id: number;
  name: string;
  run_number: number;
  status: string;
  conclusion: string | null;
  head_branch: string;
  head_sha: string;
  run_started_at?: string;
  updated_at?: string;
  actor?: { login: string };
  triggering_actor?: { login: string };
  event?: string;
};

type RunListProps = {
  runs: WorkflowRun[];
  loading: boolean;
  error: string | null;
  statusFilter: string;
  onStatusFilterChange: (value: string) => void;
  branchFilter: string;
  onBranchFilterChange: (value: string) => void;
  eventFilter: string;
  onEventFilterChange: (value: string) => void;
  page: number;
  totalCount: number;
  perPage: number;
  onPageChange: (p: number) => void;
};

function formatDuration(run: WorkflowRun): string {
  if (!run.run_started_at) return "—";
  const start = new Date(run.run_started_at).getTime();
  const end = run.updated_at ? new Date(run.updated_at).getTime() : Date.now();
  const seconds = Math.max(0, Math.floor((end - start) / 1000));
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}m ${s}s`;
}

function triggeredBy(run: WorkflowRun): string {
  const actor = run.triggering_actor?.login ?? run.actor?.login ?? "unknown";
  const event = run.event ?? "workflow";
  return `${event} by ${actor}`;
}

export function RunList({
  runs,
  loading,
  error,
  statusFilter,
  onStatusFilterChange,
  branchFilter,
  onBranchFilterChange,
  eventFilter,
  onEventFilterChange,
  page,
  totalCount,
  perPage,
  onPageChange,
}: RunListProps) {
  const totalPages = Math.max(1, Math.ceil(totalCount / perPage));
  const isFirstPage = page <= 1;
  const isLastPage = page >= totalPages;

  return (
    <div className="bg-white border border-[#d0d7de] rounded-md overflow-hidden">
      <div className="px-4 py-3 bg-[#f6f8fa] border-b border-[#d0d7de] flex flex-wrap gap-3 items-center">
        <label className="flex items-center gap-2 text-sm text-[#656d76]">
          Status
          <select
            aria-label="Filter by status"
            value={statusFilter}
            onChange={(e) => onStatusFilterChange(e.target.value)}
            className="rounded border border-[#d0d7de] bg-white px-2 py-1 text-sm text-[#1f2328]"
          >
            <option value="queued">queued</option>
            <option value="in_progress">in_progress</option>
            <option value="completed">completed</option>
            <option value="all">all</option>
          </select>
        </label>

        <label className="flex items-center gap-2 text-sm text-[#656d76]">
          Branch
          <input
            aria-label="Filter by branch"
            type="text"
            value={branchFilter}
            onChange={(e) => onBranchFilterChange(e.target.value)}
            placeholder="Branch name"
            className="rounded border border-[#d0d7de] bg-white px-2 py-1 text-sm text-[#1f2328]"
          />
        </label>

        <label className="flex items-center gap-2 text-sm text-[#656d76]">
          Event
          <select
            aria-label="Filter by event"
            value={eventFilter}
            onChange={(e) => onEventFilterChange(e.target.value)}
            className="rounded border border-[#d0d7de] bg-white px-2 py-1 text-sm text-[#1f2328]"
          >
            <option value="push">push</option>
            <option value="pull_request">pull_request</option>
            <option value="workflow_dispatch">workflow_dispatch</option>
            <option value="all">all</option>
          </select>
        </label>
      </div>

      {error && (
        <div className="mx-4 mt-4 rounded-md border border-[#cf222e] bg-[#ffebe9] px-4 py-3 text-sm text-[#cf222e]">
          {error}
        </div>
      )}

      <div className="px-4 py-3 bg-[#f6f8fa] border-b border-[#d0d7de] text-sm text-[#656d76]">
        {loading ? (
          "Loading…"
        ) : (
          <span>
            <strong>{totalCount}</strong> workflow runs
          </span>
        )}
      </div>

      {loading ? (
        <div className="px-4 py-8 text-center text-[#656d76]">Loading workflow runs…</div>
      ) : runs.length === 0 ? (
        <div className="px-4 py-8 text-center text-[#656d76]">No workflow runs found.</div>
      ) : (
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-[#d0d7de] bg-[#f6f8fa] text-left text-xs text-[#656d76]">
              <th className="px-4 py-2 font-medium">Workflow</th>
              <th className="px-4 py-2 font-medium">Run</th>
              <th className="px-4 py-2 font-medium">Status</th>
              <th className="px-4 py-2 font-medium">Branch</th>
              <th className="px-4 py-2 font-medium">Commit</th>
              <th className="px-4 py-2 font-medium">Duration</th>
              <th className="px-4 py-2 font-medium">Triggered by</th>
            </tr>
          </thead>
          <tbody>
            {runs.map((run) => (
              <tr
                key={run.id}
                className="border-b border-[#d8dee4] last:border-b-0 hover:bg-[#fafbfc]"
              >
                <td className="px-4 py-3 font-semibold text-[#1f2328]">{run.name}</td>
                <td className="px-4 py-3 font-mono text-xs">#{run.run_number}</td>
                <td className="px-4 py-3">
                  <RunStatusBadge status={run.status} conclusion={run.conclusion} />
                </td>
                <td className="px-4 py-3">
                  <span className="bg-[#ddf4ff] text-[#0969da] px-1.5 py-0.5 rounded font-mono text-[11px]">
                    {run.head_branch}
                  </span>
                </td>
                <td className="px-4 py-3 font-mono text-xs">{run.head_sha.slice(0, 7)}</td>
                <td className="px-4 py-3 text-[#656d76]">{formatDuration(run)}</td>
                <td className="px-4 py-3 text-[#656d76]">{triggeredBy(run)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <div className="flex items-center justify-between px-4 py-3 border-t border-[#d0d7de] bg-[#f6f8fa]">
        <span className="text-sm text-[#656d76]">
          Page {page} of {totalPages}
        </span>
        <div className="flex gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={isFirstPage}
            onClick={() => onPageChange(page - 1)}
          >
            Previous
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={isLastPage}
            onClick={() => onPageChange(page + 1)}
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  );
}
