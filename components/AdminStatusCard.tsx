export interface AdminStatusCardProps {
  name: string;
  status: "ok" | "error" | "unknown";
}

const statusBadgeClass: Record<AdminStatusCardProps["status"], string> = {
  ok: "bg-green-500",
  error: "bg-red-500",
  unknown: "bg-gray-400",
};

export function AdminStatusCard({ name, status }: AdminStatusCardProps) {
  return (
    <div>
      <span>{name}</span>
      <span className={statusBadgeClass[status]}>{status}</span>
    </div>
  );
}
