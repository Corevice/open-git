import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("recharts", () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="latency-chart">{children}</div>
  ),
  LineChart: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="line-chart">{children}</div>
  ),
  Line: ({ dataKey }: { dataKey: string }) => (
    <div data-testid={`line-${dataKey}`} />
  ),
  XAxis: () => null,
  YAxis: () => null,
  Tooltip: () => null,
  Legend: () => null,
}));

import { LatencyChart } from "@/components/ui/latency-chart";

describe("LatencyChart", () => {
  it("renders without crashing", () => {
    render(
      <LatencyChart
        data={[{ name: "run1", p50: 80, p95: 250, p99: 600 }]}
      />,
    );

    expect(screen.getByTestId("latency-chart")).toBeInTheDocument();
    expect(screen.getByTestId("line-chart")).toBeInTheDocument();
    expect(screen.getByTestId("line-p50")).toBeInTheDocument();
    expect(screen.getByTestId("line-p95")).toBeInTheDocument();
    expect(screen.getByTestId("line-p99")).toBeInTheDocument();
  });
});
