export interface PerfMetrics {
  p50_ms: number;
  p95_ms: number;
  p99_ms: number;
  throughput_rps: number;
  error_rate: number;
  total_requests: number;
}

export interface PerfRegressionResult {
  vs_baseline: string;
  flagged: boolean;
  delta_pct: number;
}

export interface PerfBenchmarkItem {
  id: string;
  scenario_name: string;
  environment: string;
  status: string;
  slo_result: string;
  started_at: string;
  finished_at: string | null;
  git_sha: string | null;
  metrics: PerfMetrics;
  regression: PerfRegressionResult | null;
  created_at: string;
}

export interface PerfSummaryLatest {
  scenario_name: string;
  slo_result: string;
  p95_ms: number;
  error_rate: number;
  run_at: string;
}

export interface PerfSummaryResponse {
  latest: PerfSummaryLatest[];
  slo_status: {
    overall: string;
    violations: string[];
  };
  grafana_url: string;
}

export interface PerfBenchmarksResponse {
  items: PerfBenchmarkItem[];
  next_cursor: string;
}

export interface PerfJobStatusResponse {
  status: string;
  benchmark_id: string | null;
}
