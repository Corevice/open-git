"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";

type PaginationProps = {
  page: number;
  hasNext: boolean;
  hasPrev: boolean;
  basePath: string;
};

export function Pagination({ page, hasNext, hasPrev, basePath }: PaginationProps) {
  const searchParams = useSearchParams();

  const buildPageUrl = (targetPage: number) => {
    const params = new URLSearchParams(searchParams.toString());
    if (targetPage <= 1) {
      params.delete("page");
    } else {
      params.set("page", String(targetPage));
    }
    const qs = params.toString();
    return qs ? `${basePath}?${qs}` : basePath;
  };

  return (
    <nav className="flex justify-center gap-2 py-5" aria-label="Pagination">
      {hasPrev ? (
        <Link
          href={buildPageUrl(page - 1)}
          className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm text-[#0969da] hover:bg-[#f6f8fa]"
        >
          Previous
        </Link>
      ) : (
        <span
          className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm text-[#656d76] opacity-50 cursor-not-allowed"
          aria-disabled="true"
        >
          Previous
        </span>
      )}
      {hasNext ? (
        <Link
          href={buildPageUrl(page + 1)}
          className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm text-[#0969da] hover:bg-[#f6f8fa]"
        >
          Next
        </Link>
      ) : null}
    </nav>
  );
}
