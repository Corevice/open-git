"use client";

import * as React from "react";

import { cn } from "@/lib/utils";

export type LatencyChartDatum = {
  scenario_name: string;
  p50_ms: number;
  p95_ms: number;
  p99_ms: number;
};

type LatencyChartProps = {
  data: LatencyChartDatum[];
  className?: string;
};

const SERIES: Array<{
  key: keyof Omit<LatencyChartDatum, "scenario_name">;
  label: string;
  color: string;
}> = [
  { key: "p50_ms", label: "p50", color: "bg-emerald-500" },
  { key: "p95_ms", label: "p95", color: "bg-amber-500" },
  { key: "p99_ms", label: "p99", color: "bg-rose-500" },
];

/**
 * Lightweight dependency-free latency bar chart. Renders p50/p95/p99 latency
 * for each scenario as horizontal bars scaled to the largest observed value.
 */
export function LatencyChart({ data, className }: LatencyChartProps) {
  const max = React.useMemo(() => {
    let value = 0;
    for (const datum of data) {
      value = Math.max(value, datum.p50_ms, datum.p95_ms, datum.p99_ms);
    }
    return value || 1;
  }, [data]);

  if (data.length === 0) {
    return (
      <p className="text-sm text-slate-500">No latency data available.</p>
    );
  }

  return (
    <div className={cn("space-y-4", className)}>
      <div className="flex flex-wrap gap-4 text-xs text-slate-500">
        {SERIES.map((series) => (
          <span key={series.key} className="flex items-center gap-1.5">
            <span className={cn("inline-block size-2.5 rounded-sm", series.color)} />
            {series.label}
          </span>
        ))}
      </div>
      <ul className="space-y-3">
        {data.map((datum) => (
          <li key={datum.scenario_name} className="space-y-1">
            <div className="text-xs font-medium text-slate-600">
              {datum.scenario_name}
            </div>
            <div className="space-y-1">
              {SERIES.map((series) => {
                const value = datum[series.key];
                const width = `${Math.max(2, (value / max) * 100)}%`;
                return (
                  <div key={series.key} className="flex items-center gap-2">
                    <div className="h-2.5 flex-1 overflow-hidden rounded-full bg-slate-100">
                      <div
                        className={cn("h-full rounded-full", series.color)}
                        style={{ width }}
                      />
                    </div>
                    <span className="w-16 text-right text-xs tabular-nums text-slate-500">
                      {value} ms
                    </span>
                  </div>
                );
              })}
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
