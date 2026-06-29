import { SkeletonBlock } from "@/components/ui/skeleton";

export default function Loading() {
  return (
    <div className="space-y-3 p-6">
      {Array.from({ length: 5 }).map((_, i) => (
        <SkeletonBlock key={i} className="h-16 w-full" />
      ))}
    </div>
  );
}
