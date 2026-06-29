import Link from "next/link";
import type { ReactNode } from "react";

export function RepoPageSkeleton({ className }: { className?: string }) {
  return (
    <div className={`animate-pulse rounded bg-[#eaeef2] ${className ?? ""}`} />
  );
}

export function RepoPageLoadingShell({
  owner,
  children,
}: {
  owner: string;
  children: ReactNode;
}) {
  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <header className="h-16 bg-white/85 backdrop-blur border-b border-[color:var(--border)] sticky top-0 z-[100]">
        <div className="max-w-[1280px] mx-auto px-6 flex items-center justify-between h-full">
          <Link href="/dashboard" className="text-lg font-extrabold flex items-center gap-2">
            <span>🐙</span> OpenHub
          </Link>
          <Link
            href="/dashboard"
            className="px-2 py-1 rounded-full text-xs font-medium bg-[color:var(--primary-light)] text-[color:var(--primary)]"
          >
            {owner}
          </Link>
        </div>
      </header>

      <div className="max-w-[1280px] mx-auto px-6 py-6">{children}</div>
    </div>
  );
}
