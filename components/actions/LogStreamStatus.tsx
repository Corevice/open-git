"use client";

type LogStreamStatusProps = {
  status: "streaming" | "completed" | "reconnecting" | "failed";
};

const STATUS_CONFIG: Record<
  LogStreamStatusProps["status"],
  { label: string; dotClass: string; textClass: string; spin?: boolean }
> = {
  streaming: {
    label: "Streaming",
    dotClass: "bg-[#0969da]",
    textClass: "text-[#0969da]",
    spin: true,
  },
  completed: {
    label: "Completed",
    dotClass: "bg-[#1a7f37]",
    textClass: "text-[#1a7f37]",
  },
  reconnecting: {
    label: "Reconnecting",
    dotClass: "bg-[#bf8700]",
    textClass: "text-[#bf8700]",
  },
  failed: {
    label: "Failed",
    dotClass: "bg-[#cf222e]",
    textClass: "text-[#cf222e]",
  },
};

export default function LogStreamStatus({ status }: LogStreamStatusProps) {
  const config = STATUS_CONFIG[status];

  return (
    <div
      role="status"
      aria-label={config.label}
      className={`inline-flex items-center gap-2 text-xs ${config.textClass}`}
    >
      <span
        className={
          config.spin
            ? "inline-block h-2.5 w-2.5 shrink-0 animate-spin rounded-full border-2 border-[#d0d7de] border-t-[#0969da]"
            : `inline-block h-2.5 w-2.5 shrink-0 rounded-full ${config.dotClass}`
        }
        aria-hidden
      />
      <span>{config.label}</span>
    </div>
  );
}
