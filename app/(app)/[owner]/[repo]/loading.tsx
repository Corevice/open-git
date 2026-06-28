import { SkeletonBlock } from "@/components/ui/skeleton";

export default function Loading() {
  return (
    <div className="space-y-6 p-6">
      <div className="space-y-2">
        <SkeletonBlock className="h-8 w-full" />
        <SkeletonBlock className="h-8 w-full" />
      </div>
      <SkeletonBlock className="h-64 w-full" />
    </div>
  );
}
