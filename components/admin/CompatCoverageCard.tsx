import type { CompatCoverage } from "@/app/admin/compatibility/types";

interface CompatCoverageCardProps {
  coverage: CompatCoverage;
  generatedAt: string;
}

function formatGeneratedAt(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return date.toLocaleString(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  });
}

export function CompatCoverageCard({
  coverage,
  generatedAt,
}: CompatCoverageCardProps) {
  const ratePercent = `${(coverage.rate * 100).toFixed(1)}%`;

  const stats = [
    { label: "Total", value: coverage.total_endpoints, color: "text-[#24292f]" },
    { label: "Passing", value: coverage.passing, color: "text-[#1a7f37]" },
    { label: "Failing", value: coverage.failing, color: "text-[#cf222e]" },
    {
      label: "Unimplemented",
      value: coverage.unimplemented,
      color: "text-[#656d76]",
    },
  ] as const;

  return (
    <div className="mb-6 overflow-hidden rounded-md border border-[#d0d7de] bg-white">
      <div className="border-b border-[#d0d7de] px-6 py-4">
        <h2 className="text-lg font-semibold text-[#24292f]">
          Coverage Summary
        </h2>
        <p className="mt-1 text-sm text-[#656d76]">
          Last generated: {formatGeneratedAt(generatedAt)}
        </p>
      </div>
      <div className="grid grid-cols-2 gap-4 px-6 py-5 sm:grid-cols-4">
        {stats.map((stat) => (
          <div key={stat.label}>
            <p className="text-xs font-medium uppercase tracking-wide text-[#656d76]">
              {stat.label}
            </p>
            <p className={`mt-1 text-2xl font-semibold ${stat.color}`}>
              {stat.value}
            </p>
          </div>
        ))}
      </div>
      <div className="border-t border-[#d0d7de] bg-[#f6f8fa] px-6 py-4">
        <p className="text-sm text-[#656d76]">
          Coverage rate:{" "}
          <span className="text-lg font-semibold text-[#0969da]">
            {ratePercent}
          </span>
        </p>
      </div>
    </div>
  );
}
