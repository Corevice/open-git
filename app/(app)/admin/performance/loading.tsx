import { Card, CardContent, CardHeader } from "@/components/ui/card";

function SkeletonBlock({ className }: { className?: string }) {
  return (
    <div
      className={`animate-pulse rounded-md bg-slate-200 ${className ?? ""}`}
    />
  );
}

export default function PerformanceLoading() {
  return (
    <div className="space-y-6 p-6">
      <SkeletonBlock className="h-8 w-64" />

      <Card>
        <CardHeader>
          <SkeletonBlock className="h-6 w-32" />
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          <SkeletonBlock className="h-6 w-28 rounded-full" />
          <SkeletonBlock className="h-6 w-28 rounded-full" />
          <SkeletonBlock className="h-6 w-28 rounded-full" />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <SkeletonBlock className="h-6 w-40" />
        </CardHeader>
        <CardContent>
          <SkeletonBlock className="h-64 w-full" />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <SkeletonBlock className="h-6 w-48" />
        </CardHeader>
        <CardContent className="space-y-3">
          <SkeletonBlock className="h-10 w-full" />
          <SkeletonBlock className="h-10 w-full" />
          <SkeletonBlock className="h-10 w-full" />
          <SkeletonBlock className="h-10 w-full" />
        </CardContent>
      </Card>
    </div>
  );
}
