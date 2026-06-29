import { cn } from "@/lib/utils";

export function SkeletonBlock({ className }: { className?: string }) {
  return (
    <div className={cn("animate-pulse bg-[#eaeef2] rounded", className)} />
  );
}

export function SkeletonText({ className }: { className?: string }) {
  return <SkeletonBlock className={cn("h-4 w-3/4", className)} />;
}
