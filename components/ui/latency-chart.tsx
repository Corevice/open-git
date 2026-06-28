"use client";

import {
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

export interface LatencyChartProps {
  data: Array<{ name: string; p50: number; p95: number; p99: number }>;
}

export function LatencyChart({ data }: LatencyChartProps) {
  return (
    <ResponsiveContainer width="100%" height={300}>
      <LineChart data={data}>
        <XAxis dataKey="name" />
        <YAxis unit="ms" />
        <Tooltip />
        <Legend />
        <Line type="monotone" dataKey="p50" stroke="#3b82f6" name="p50" />
        <Line type="monotone" dataKey="p95" stroke="#f59e0b" name="p95" />
        <Line type="monotone" dataKey="p99" stroke="#ef4444" name="p99" />
      </LineChart>
    </ResponsiveContainer>
  );
}
