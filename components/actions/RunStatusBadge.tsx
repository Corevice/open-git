"use client";

type StatusConfig = {
  label: string;
  className: string;
};

const statusConfig: Record<string, StatusConfig> = {
  queued: { label: "Queued", className: "bg-[#fff8c5] text-[#9a6700]" },
  in_progress: { label: "In progress", className: "bg-[#fff8c5] text-[#9a6700]" },
  "completed:success": { label: "Success", className: "bg-[#dafbe1] text-[#1a7f37]" },
  "completed:failure": { label: "Failure", className: "bg-[#ffebe9] text-[#cf222e]" },
  "completed:cancelled": { label: "Cancelled", className: "bg-[#eaeef2] text-[#656d76]" },
  "completed:skipped": { label: "Skipped", className: "bg-[#eaeef2] text-[#656d76]" },
  "completed:timed_out": { label: "Timed out", className: "bg-[#eaeef2] text-[#656d76]" },
};

function resolveStatusConfig(status: string, conclusion: string | null): StatusConfig {
  if (status === "queued" || status === "in_progress") {
    return statusConfig[status];
  }
  if (conclusion) {
    const key = `completed:${conclusion}`;
    if (statusConfig[key]) {
      return statusConfig[key];
    }
    return {
      label: conclusion.charAt(0).toUpperCase() + conclusion.slice(1),
      className: "bg-[#eaeef2] text-[#656d76]",
    };
  }
  return {
    label: status.charAt(0).toUpperCase() + status.slice(1),
    className: "bg-[#eaeef2] text-[#656d76]",
  };
}

type RunStatusBadgeProps = {
  status: string;
  conclusion: string | null;
};

export function RunStatusBadge({ status, conclusion }: RunStatusBadgeProps) {
  const config = resolveStatusConfig(status, conclusion);

  return (
    <span
      className={`inline-block px-2 py-0.5 rounded-full text-xs ${config.className}`}
    >
      {config.label}
    </span>
  );
}
