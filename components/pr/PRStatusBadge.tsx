import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

type PRStatusBadgeProps = {
  state: "open" | "closed" | "merged";
};

const stateConfig: Record<
  PRStatusBadgeProps["state"],
  { label: string; className: string }
> = {
  open: { label: "Open", className: "border-transparent bg-[#dafbe1] text-[#1a7f37]" },
  merged: { label: "Merged", className: "border-transparent bg-[#8250df] text-white" },
  closed: { label: "Closed", className: "border-transparent bg-[#ffebe9] text-[#cf222e]" },
};

export function PRStatusBadge({ state }: PRStatusBadgeProps) {
  const config = stateConfig[state];

  return (
    <Badge className={cn(config.className)}>{config.label}</Badge>
  );
}
