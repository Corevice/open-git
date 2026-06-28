"use client";

import Link from "next/link";

import { cn } from "@/lib/utils";

export interface PaginationProps {
  page: number;
  hasNext: boolean;
  hasPrev: boolean;
  basePath: string;
}

export function Pagination({
  page,
  hasNext,
  hasPrev,
  basePath,
}: PaginationProps) {
  return (
    <nav className="flex items-center gap-2" aria-label="Pagination">
      <Link
        href={`${basePath}?page=${page - 1}`}
        aria-disabled={!hasPrev}
        className={cn(
          "inline-flex h-9 items-center justify-center rounded-md border border-slate-300 bg-white px-4 text-sm font-medium hover:bg-slate-100",
          !hasPrev && "pointer-events-none opacity-50",
        )}
      >
        Previous
      </Link>
      <Link
        href={`${basePath}?page=${page + 1}`}
        aria-disabled={!hasNext}
        className={cn(
          "inline-flex h-9 items-center justify-center rounded-md border border-slate-300 bg-white px-4 text-sm font-medium hover:bg-slate-100",
          !hasNext && "pointer-events-none opacity-50",
        )}
      >
        Next
      </Link>
    </nav>
  );
}
