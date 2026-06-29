"use client";

import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { RunList, type WorkflowRun } from "@/components/actions/RunList";
import { listRuns } from "@/lib/api/actions";

export default function ActionsPage() {
  const params = useParams();
  const owner = params.owner as string;
  const repo = params.repo as string;

  const [runs, setRuns] = useState<WorkflowRun[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState("");
  const [branchFilter, setBranchFilter] = useState("");
  const [eventFilter, setEventFilter] = useState("");
  const [page, setPage] = useState(1);
  const [perPage] = useState(30);

  const loadRuns = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await listRuns(owner, repo, {
        status: statusFilter && statusFilter !== "all" ? statusFilter : undefined,
        branch: branchFilter || undefined,
        event: eventFilter && eventFilter !== "all" ? eventFilter : undefined,
        page,
        per_page: perPage,
      });
      setRuns(data.workflow_runs ?? []);
      setTotalCount(data.total_count ?? 0);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load workflow runs");
    } finally {
      setLoading(false);
    }
  }, [owner, repo, statusFilter, branchFilter, eventFilter, page, perPage]);

  useEffect(() => {
    loadRuns();
  }, [loadRuns]);

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <div className="max-w-[1280px] mx-auto px-6 py-6">
        <h1 className="text-2xl font-semibold mb-6">
          <span className="text-[#0969da]">{owner}</span> /{" "}
          <span className="text-[#0969da]">{repo}</span>
          <span className="ml-2 text-lg font-normal text-[#656d76]">Actions</span>
        </h1>

        <RunList
          runs={runs}
          loading={loading}
          error={error}
          statusFilter={statusFilter}
          onStatusFilterChange={setStatusFilter}
          branchFilter={branchFilter}
          onBranchFilterChange={setBranchFilter}
          eventFilter={eventFilter}
          onEventFilterChange={setEventFilter}
          page={page}
          totalCount={totalCount}
          perPage={perPage}
          onPageChange={setPage}
        />
      </div>
    </div>
  );
}
