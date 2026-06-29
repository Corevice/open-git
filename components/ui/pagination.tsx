"use client";

import Link from "next/link";
import type { MouseEvent } from "react";

import { cn } from "@/lib/utils";

export interface PaginationProps {
  page: number;
  hasNext: boolean;
  hasPrev: boolean;
  basePath: string;
}

function sanitizeBasePath(path: string): string {
  if (!path.startsWith("/") || path.startsWith("//") || path.includes("://")) {
    return "/";
  }
  return path;
}

function preventDisabledNavigation(
  event: MouseEvent<HTMLAnchorElement>,
  enabled: boolean,
): void {
  if (!enabled) {
    event.preventDefault();
  }
}

export function Pagination({
  page,
  hasNext,
  hasPrev,
  basePath,
}: PaginationProps) {
  const safeBasePath = sanitizeBasePath(basePath);

  return (
    <nav className="flex items-center gap-2" aria-label="Pagination">
      <Link
        href={`${safeBasePath}?page=${page - 1}`}
        aria-disabled={hasPrev ? undefined : "true"}
        tabIndex={hasPrev ? undefined : -1}
        onClick={(event) => preventDisabledNavigation(event, hasPrev)}
        className={cn(
          "inline-flex h-9 items-center justify-center rounded-md border border-slate-300 bg-white px-4 text-sm font-medium hover:bg-slate-100",
          !hasPrev && "pointer-events-none opacity-50",
        )}
      >
        Previous
      </Link>
      <span
        className="inline-flex h-9 items-center justify-center px-2 text-sm text-slate-600"
        aria-current="page"
      >
        Page {page}
      </span>
      <Link
        href={`${safeBasePath}?page=${page + 1}`}
        aria-disabled={hasNext ? undefined : "true"}
        tabIndex={hasNext ? undefined : -1}
        onClick={(event) => preventDisabledNavigation(event, hasNext)}
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
