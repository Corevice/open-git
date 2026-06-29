import { redirect } from "next/navigation";

import { getBenchmarks, getPerformanceSummary } from "@/lib/api/perf";
import { getServerSession } from "@/lib/session";

import { PerformanceClient } from "./PerformanceClient";

export default async function PerformancePage() {
  const session = await getServerSession();

  if (session?.user?.role !== "admin") {
    redirect("/403");
  }

  const [summary, benchmarks] = await Promise.all([
    getPerformanceSummary(),
    getBenchmarks({ limit: 20 }),
  ]);

  return (
    <div className="space-y-6 p-6">
      <h1 className="text-2xl font-semibold">Performance Dashboard</h1>
      <PerformanceClient
        initialSummary={summary}
        initialBenchmarks={benchmarks}
      />
    </div>
  );
}
