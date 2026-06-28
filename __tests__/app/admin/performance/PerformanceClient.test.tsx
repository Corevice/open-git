import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PerformanceClient } from "@/app/(app)/admin/performance/PerformanceClient";
import { ApiError } from "@/lib/api";
import type {
  PerfBenchmarksResponse,
  PerfSummaryResponse,
} from "@/lib/api/perf";

vi.mock("@/components/ui/chart", () => ({
  LatencyChart: () => <div data-testid="latency-chart" />,
}));

const getBenchmarks = vi.fn();
const getJobStatus = vi.fn();
const runBenchmark = vi.fn();

vi.mock("@/lib/api/perf", () => ({
  getBenchmarks: (...args: unknown[]) => getBenchmarks(...args),
  getJobStatus: (...args: unknown[]) => getJobStatus(...args),
  runBenchmark: (...args: unknown[]) => runBenchmark(...args),
}));

const mockSummary: PerfSummaryResponse = {
  latest: [
    {
      scenario_name: "rest-repos-read",
      slo_result: "pass",
      p95_ms: 180,
      error_rate: 0.001,
      run_at: "2025-06-01T12:00:00Z",
    },
    {
      scenario_name: "graphql-viewer",
      slo_result: "fail",
      p95_ms: 920,
      error_rate: 0.02,
      run_at: "2025-06-01T12:00:00Z",
    },
  ],
  slo_status: {
    overall: "fail",
    violations: ["graphql-viewer"],
  },
  grafana_url: "https://grafana.example/d/perf",
};

const mockBenchmarks: PerfBenchmarksResponse = {
  items: [
    {
      id: "bench-1",
      scenario_name: "rest-repos-read",
      environment: "ci",
      status: "completed",
      slo_result: "pass",
      started_at: "2025-06-01T11:55:00Z",
      finished_at: "2025-06-01T12:00:00Z",
      git_sha: "abc1234",
      metrics: {
        p50_ms: 80,
        p95_ms: 180,
        p99_ms: 420,
        throughput_rps: 1200,
        error_rate: 0.001,
        total_requests: 360000,
      },
      regression: {
        vs_baseline: "+3%",
        flagged: false,
        delta_pct: 3,
      },
    },
    {
      id: "bench-2",
      scenario_name: "graphql-viewer",
      environment: "docker-compose",
      status: "completed",
      slo_result: "fail",
      started_at: "2025-06-01T11:50:00Z",
      finished_at: "2025-06-01T11:55:00Z",
      git_sha: "def5678",
      metrics: {
        p50_ms: 200,
        p95_ms: 920,
        p99_ms: 1400,
        throughput_rps: 800,
        error_rate: 0.02,
        total_requests: 240000,
      },
    },
  ],
  next_cursor: null,
};

function renderClient(
  summary: PerfSummaryResponse = mockSummary,
  benchmarks: PerfBenchmarksResponse = mockBenchmarks,
) {
  return render(
    <PerformanceClient
      initialSummary={summary}
      initialBenchmarks={benchmarks}
    />,
  );
}

describe("PerformanceClient", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders SLO badge with slo_result='pass' for mock data", () => {
    renderClient();

    expect(
      screen.getByText("rest-repos-read: pass"),
    ).toBeInTheDocument();
  });

  it("renders SLO badge with slo_result='fail' styled as destructive", () => {
    renderClient();

    const failBadge = screen.getByText("graphql-viewer: fail");
    expect(failBadge).toBeInTheDocument();
    expect(failBadge.className).toMatch(/bg-red-600/);
  });

  it("clicking Run Benchmark calls runBenchmark() mock", async () => {
    const user = userEvent.setup();
    runBenchmark.mockResolvedValue({
      job_id: "job-1",
      status: "queued",
    });
    getJobStatus.mockResolvedValue({ status: "running", benchmark_id: null });

    renderClient();

    await user.click(screen.getByRole("button", { name: "Run Benchmark" }));

    await waitFor(() => {
      expect(runBenchmark).toHaveBeenCalledTimes(1);
    });
  });

  it("when runBenchmark resolves with 409 error, shows conflict message text", async () => {
    const user = userEvent.setup();
    runBenchmark.mockRejectedValue(
      new ApiError(409, "Benchmark already running"),
    );

    renderClient();

    await user.click(screen.getByRole("button", { name: "Run Benchmark" }));

    expect(
      await screen.findByText("Benchmark already running"),
    ).toBeInTheDocument();
  });

  it("benchmark table rows render scenario_name from mock data", () => {
    renderClient();

    expect(screen.getByText("rest-repos-read")).toBeInTheDocument();
    expect(screen.getByText("graphql-viewer")).toBeInTheDocument();
  });
});
