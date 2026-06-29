import type { ActionCompatibilityStatus } from "@/lib/api-types";

type CompatBadgeProps = {
  status: ActionCompatibilityStatus | string;
};

const STATUS_STYLES: Record<string, { label: string; className: string }> = {
  supported: { label: "Supported", className: "bg-[#dafbe1] text-[#1a7f37]" },
  partial: { label: "Partial", className: "bg-[#fff8c5] text-[#9a6700]" },
  unsupported: { label: "Unsupported", className: "bg-[#ffebe9] text-[#cf222e]" },
  unknown: { label: "Unknown", className: "bg-[#eaeef2] text-[#656d76]" },
};

export function CompatBadge({ status }: CompatBadgeProps) {
  const config =
    STATUS_STYLES[status] ?? {
      label: status,
      className: "bg-[#eaeef2] text-[#656d76]",
    };

  return (
    <span
      className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${config.className}`}
    >
      {config.label}
    </span>
  );
}
