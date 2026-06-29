"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { LatencyChart } from "@/components/ui/chart";
import {
  TableRoot as Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ApiError } from "@/lib/api";
import {
  getBenchmarks,
  getJobStatus,
  runBenchmark,
  type PerfBenchmarkItem,
  type PerfBenchmarksResponse,
  type PerfSummaryResponse,
} from "@/lib/api/perf";

type PerformanceClientProps = {
  initialSummary: PerfSummaryResponse;
  initialBenchmarks: PerfBenchmarksResponse;
};

function formatRunAt(value: string | undefined): string {
  if (!value) {
    return "—";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

function sloBadgeVariant(
  sloResult: string,
): "default" | "destructive" {
  return sloResult === "pass" ? "default" : "destructive";
}

function benchmarkRunAt(item: PerfBenchmarkItem): string {
  return item.run_at ?? item.finished_at ?? item.started_at;
}

export function PerformanceClient({
  initialSummary,
  initialBenchmarks,
}: PerformanceClientProps) {
  const [benchmarks, setBenchmarks] = useState(initialBenchmarks.items);
  const [nextCursor, setNextCursor] = useState<string | null>(
    initialBenchmarks.next_cursor ?? null,
  );
  const [selectedBenchmark, setSelectedBenchmark] =
    useState<PerfBenchmarkItem | null>(null);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const [jobStatus, setJobStatus] = useState<string | null>(null);
  const [, setJobId] = useState<string | null>(null);
  const [isRunning, setIsRunning] = useState(false);
  const [conflictMessage, setConflictMessage] = useState<string | null>(null);
  const [loadingMore, setLoadingMore] = useState(false);
  const pollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const chartData = useMemo(
    () =>
      benchmarks.map((item) => ({
        scenario_name: item.scenario_name,
        p50_ms: item.metrics.p50_ms,
        p95_ms: item.metrics.p95_ms,
        p99_ms: item.metrics.p99_ms,
      })),
    [benchmarks],
  );

  const clearPolling = useCallback(() => {
    if (pollIntervalRef.current) {
      clearInterval(pollIntervalRef.current);
      pollIntervalRef.current = null;
    }
  }, []);

  const refreshBenchmarks = useCallback(async () => {
    const refreshed = await getBenchmarks({ limit: 20 });
    setBenchmarks(refreshed.items);
    setNextCursor(refreshed.next_cursor ?? null);
  }, []);

  const startJobPolling = useCallback(
    (activeJobId: string) => {
      clearPolling();
      pollIntervalRef.current = setInterval(async () => {
        try {
          const status = await getJobStatus(activeJobId);
          setJobStatus(status.status);

          if (
            status.status === "completed" ||
            status.status === "failed" ||
            status.status === "timeout"
          ) {
            clearPolling();
            setIsRunning(false);
            await refreshBenchmarks();
          }
        } catch {
          clearPolling();
          setIsRunning(false);
        }
      }, 5000);
    },
    [clearPolling, refreshBenchmarks],
  );

  useEffect(() => {
    return () => {
      clearPolling();
    };
  }, [clearPolling]);

  const handleLoadMore = async () => {
    if (!nextCursor || loadingMore) {
      return;
    }

    setLoadingMore(true);
    try {
      const response = await getBenchmarks({ cursor: nextCursor });
      setBenchmarks((current) => [...current, ...response.items]);
      setNextCursor(response.next_cursor ?? null);
    } finally {
      setLoadingMore(false);
    }
  };

  const handleRunBenchmark = async () => {
    setConflictMessage(null);

    try {
      const response = await runBenchmark();
      setJobId(response.job_id);
      setJobStatus(response.status);
      setIsRunning(true);
      startJobPolling(response.job_id);
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) {
        setConflictMessage(
          error.message || "A benchmark job is already running.",
        );
        return;
      }
      throw error;
    }
  };

  const openDrawer = (item: PerfBenchmarkItem) => {
    setSelectedBenchmark(item);
    setIsDrawerOpen(true);
  };

  const closeDrawer = () => {
    setIsDrawerOpen(false);
    setSelectedBenchmark(null);
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>SLO Status</CardTitle>
          <Badge
            variant={sloBadgeVariant(initialSummary.slo_status.overall)}
          >
            Overall: {initialSummary.slo_status.overall}
          </Badge>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          {initialSummary.latest.map((item) => (
            <Badge
              key={item.scenario_name}
              variant={sloBadgeVariant(item.slo_result)}
            >
              {item.scenario_name}: {item.slo_result}
            </Badge>
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Latency Overview</CardTitle>
        </CardHeader>
        <CardContent>
          <LatencyChart data={chartData} />
        </CardContent>
      </Card>

      <div className="flex flex-wrap items-center gap-3">
        {initialSummary.grafana_url ? (
          <Button asChild>
            <a href={initialSummary.grafana_url} target="_blank" rel="noreferrer">
              Open Grafana
            </a>
          </Button>
        ) : null}
        <Button onClick={handleRunBenchmark} disabled={isRunning}>
          Run Benchmark
        </Button>
        {isRunning && jobStatus ? (
          <span className="text-sm text-slate-600">Job status: {jobStatus}</span>
        ) : null}
      </div>

      {conflictMessage ? (
        <p className="text-sm text-red-600">{conflictMessage}</p>
      ) : null}

      <Card>
        <CardHeader>
          <CardTitle>Benchmark History</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Scenario</TableHead>
                <TableHead>Environment</TableHead>
                <TableHead>Run At</TableHead>
                <TableHead>SLO</TableHead>
                <TableHead>P95 (ms)</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {benchmarks.map((item) => (
                <TableRow
                  key={item.id}
                  className="cursor-pointer"
                  onClick={() => openDrawer(item)}
                >
                  <TableCell>{item.scenario_name}</TableCell>
                  <TableCell>{item.environment}</TableCell>
                  <TableCell>{formatRunAt(benchmarkRunAt(item))}</TableCell>
                  <TableCell>
                    <Badge variant={sloBadgeVariant(item.slo_result)}>
                      {item.slo_result}
                    </Badge>
                  </TableCell>
                  <TableCell>{item.metrics.p95_ms}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          {nextCursor ? (
            <div className="mt-4">
              <Button
                variant="outline"
                onClick={handleLoadMore}
                disabled={loadingMore}
              >
                Load More
              </Button>
            </div>
          ) : null}
        </CardContent>
      </Card>

      {isDrawerOpen && selectedBenchmark ? (
        <dialog
          open
          className="fixed inset-y-0 right-0 z-50 m-0 h-full w-full max-w-md border-l border-slate-200 bg-white p-6 shadow-xl"
        >
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold">
              {selectedBenchmark.scenario_name}
            </h2>
            <Button variant="ghost" size="sm" onClick={closeDrawer}>
              Close
            </Button>
          </div>

          <dl className="space-y-3 text-sm">
            <div>
              <dt className="font-medium text-slate-500">Environment</dt>
              <dd>{selectedBenchmark.environment}</dd>
            </div>
            <div>
              <dt className="font-medium text-slate-500">Status</dt>
              <dd>{selectedBenchmark.status}</dd>
            </div>
            <div>
              <dt className="font-medium text-slate-500">SLO Result</dt>
              <dd>
                <Badge variant={sloBadgeVariant(selectedBenchmark.slo_result)}>
                  {selectedBenchmark.slo_result}
                </Badge>
              </dd>
            </div>
            <div>
              <dt className="font-medium text-slate-500">Git SHA</dt>
              <dd>{selectedBenchmark.git_sha ?? "—"}</dd>
            </div>
            <div>
              <dt className="font-medium text-slate-500">Started At</dt>
              <dd>{formatRunAt(selectedBenchmark.started_at)}</dd>
            </div>
            <div>
              <dt className="font-medium text-slate-500">Finished At</dt>
              <dd>{formatRunAt(selectedBenchmark.finished_at)}</dd>
            </div>
            <div>
              <dt className="font-medium text-slate-500">Metrics</dt>
              <dd className="mt-1 space-y-1">
                <p>P50: {selectedBenchmark.metrics.p50_ms} ms</p>
                <p>P95: {selectedBenchmark.metrics.p95_ms} ms</p>
                <p>P99: {selectedBenchmark.metrics.p99_ms} ms</p>
                <p>
                  Throughput: {selectedBenchmark.metrics.throughput_rps} req/s
                </p>
                <p>
                  Error rate:{" "}
                  {(selectedBenchmark.metrics.error_rate * 100).toFixed(2)}%
                </p>
                <p>
                  Total requests: {selectedBenchmark.metrics.total_requests}
                </p>
              </dd>
            </div>
            {selectedBenchmark.regression ? (
              <div>
                <dt className="font-medium text-slate-500">Regression</dt>
                <dd className="mt-1 space-y-1">
                  <p>vs baseline: {selectedBenchmark.regression.vs_baseline}</p>
                  <p>
                    flagged:{" "}
                    {selectedBenchmark.regression.flagged ? "yes" : "no"}
                  </p>
                  {selectedBenchmark.regression.delta_pct !== undefined ? (
                    <p>delta: {selectedBenchmark.regression.delta_pct}%</p>
                  ) : null}
                </dd>
              </div>
            ) : null}
          </dl>
        </dialog>
      ) : null}
    </div>
  );
}
