interface StatusBadgeProps {
  status: string;
  conclusion?: string;
  size?: "sm" | "md";
}

function getBadgeConfig(
  status: string,
  conclusion?: string,
): { label: string; className: string } {
  if (status === "queued") {
    return { label: "Queued", className: "bg-gray-400 text-white" };
  }
  if (status === "in_progress") {
    return {
      label: "In progress",
      className: "bg-yellow-400 animate-pulse text-gray-900",
    };
  }
  if (status === "completed") {
    switch (conclusion) {
      case "success":
        return { label: "Success", className: "bg-green-500 text-white" };
      case "failure":
        return { label: "Failure", className: "bg-red-500 text-white" };
      case "cancelled":
        return { label: "Cancelled", className: "bg-gray-400 text-white" };
      case "skipped":
        return { label: "Skipped", className: "bg-gray-300 text-gray-800" };
      default:
        return { label: "Completed", className: "bg-gray-400 text-white" };
    }
  }
  return { label: status, className: "bg-gray-400 text-white" };
}

export default function StatusBadge({
  status,
  conclusion,
  size = "sm",
}: StatusBadgeProps) {
  const { label, className } = getBadgeConfig(status, conclusion);
  const sizeClass =
    size === "md" ? "text-sm px-2.5 py-1" : "text-xs px-2 py-0.5";

  return (
    <span
      className={`inline-block rounded-full font-medium ${sizeClass} ${className}`}
    >
      {label}
    </span>
  );
}
