"use client";

import Link from "next/link";
import BranchSelector from "@/components/repo/BranchSelector";

interface RepoHeaderProps {
  owner: string;
  repo: string;
  isPrivate: boolean;
  defaultBranch: string;
  currentRef: string;
  onRefChange?: (ref: string) => void;
}

function buildBranches(defaultBranch: string, currentRef: string): { name: string }[] {
  if (currentRef === defaultBranch) {
    return [{ name: defaultBranch }];
  }
  return [{ name: defaultBranch }, { name: currentRef }];
}

export default function RepoHeader({
  owner,
  repo,
  isPrivate,
  defaultBranch,
  currentRef,
  onRefChange,
}: RepoHeaderProps) {
  const branches = buildBranches(defaultBranch, currentRef);

  return (
    <div className="flex flex-wrap items-center gap-3">
      <h1 className="text-2xl font-semibold m-0">
        <Link href={`/${owner}`} className="text-[#0969da] no-underline hover:underline">
          {owner}
        </Link>
        <span className="text-[#57606a]"> / </span>
        <Link
          href={`/${owner}/${repo}`}
          className="text-[#0969da] no-underline hover:underline"
        >
          {repo}
        </Link>
      </h1>
      <span
        className={
          isPrivate
            ? "px-2 py-0.5 rounded-full text-xs font-medium bg-[#eaeef2] text-[#57606a]"
            : "px-2 py-0.5 rounded-full text-xs font-medium bg-[#ddf4ff] text-[#0969da]"
        }
      >
        {isPrivate ? "Private" : "Public"}
      </span>
      <BranchSelector
        branches={branches}
        currentBranch={currentRef}
        onChange={onRefChange ?? (() => {})}
      />
    </div>
  );
}
