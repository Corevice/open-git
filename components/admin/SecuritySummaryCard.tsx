import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

interface SecuritySummaryCardProps {
  severity: string;
  count: number;
  label: string;
}

const severityBadgeClass: Record<string, string> = {
  critical: "border-transparent bg-[#cf222e] text-white",
  high: "border-transparent bg-[#ffebe9] text-[#cf222e]",
  medium: "border-transparent bg-[#fff8c5] text-[#9a6700]",
  low: "border-transparent bg-[#eaeef2] text-[#656d76]",
};

function resolveSeverityBadgeClass(severity: string): string {
  return (
    severityBadgeClass[severity.toLowerCase()] ??
    "border-transparent bg-[#eaeef2] text-[#656d76]"
  );
}

export function SecuritySummaryCard({
  severity,
  count,
  label,
}: SecuritySummaryCardProps) {
  const normalizedSeverity = severity.toLowerCase();

  return (
    <div className="overflow-hidden rounded-md border border-[#d0d7de] bg-white px-6 py-5">
      <p className="text-xs font-medium uppercase tracking-wide text-[#656d76]">
        {label}
      </p>
      <div className="mt-2 flex items-center gap-3">
        <p className="text-2xl font-semibold text-[#24292f]">{count}</p>
        <Badge
          className={cn(
            "capitalize",
            resolveSeverityBadgeClass(normalizedSeverity),
          )}
        >
          {normalizedSeverity}
        </Badge>
      </div>
    </div>
  );
}
