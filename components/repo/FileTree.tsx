"use client";

import Link from "next/link";
import { useState } from "react";

import {
  apiClient as defaultApiClient,
  createRepoApiClient,
} from "@/lib/api-client";

export interface TreeEntry {
  name: string;
  path: string;
  type: "dir" | "file";
  sha?: string;
  size?: number;
  commit_message?: string;
  committed_at?: string;
}

interface FileTreeProps {
  entries: TreeEntry[];
  owner: string;
  repo: string;
  treeRef: string;
  apiClient?: Pick<ReturnType<typeof createRepoApiClient>, "getContents">;
}

function formatRelativeTime(dateStr?: string): string {
  if (!dateStr) return "";
  const then = new Date(dateStr).getTime();
  if (Number.isNaN(then)) return "";
  const seconds = Math.floor((Date.now() - then) / 1000);
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} minute${minutes === 1 ? "" : "s"} ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} hour${hours === 1 ? "" : "s"} ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days} day${days === 1 ? "" : "s"} ago`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months} month${months === 1 ? "" : "s"} ago`;
  const years = Math.floor(months / 12);
  return `${years} year${years === 1 ? "" : "s"} ago`;
}

function blobHref(
  entry: TreeEntry,
  owner: string,
  repo: string,
  treeRef: string,
): string {
  return `/${owner}/${repo}/blob/${treeRef}/${entry.path}`;
}

export default function FileTree({
  entries,
  owner,
  repo,
  treeRef,
  apiClient = defaultApiClient,
}: FileTreeProps) {
  const [expandedDirs, setExpandedDirs] = useState<Record<string, TreeEntry[]>>(
    {},
  );
  const [loadingDirs, setLoadingDirs] = useState<Set<string>>(new Set());

  const sorted = [...entries].sort((a, b) => {
    if (a.type !== b.type) return a.type === "dir" ? -1 : 1;
    return a.name.localeCompare(b.name);
  });

  async function handleDirClick(entry: TreeEntry) {
    if (expandedDirs[entry.path]) {
      setExpandedDirs((prev) => {
        const next = { ...prev };
        delete next[entry.path];
        return next;
      });
      return;
    }

    setLoadingDirs((prev) => new Set(prev).add(entry.path));
    try {
      const children = (await apiClient.getContents(
        owner,
        repo,
        entry.path,
        treeRef,
      )) as TreeEntry[];
      setExpandedDirs((prev) => ({ ...prev, [entry.path]: children }));
    } finally {
      setLoadingDirs((prev) => {
        const next = new Set(prev);
        next.delete(entry.path);
        return next;
      });
    }
  }

  if (sorted.length === 0) {
    return (
      <p className="px-4 py-6 text-sm text-[#57606a]">This directory is empty.</p>
    );
  }

  return (
    <ul className="list-none p-0 m-0">
      {sorted.map((entry) => (
        <li key={entry.path}>
          {entry.type === "dir" ? (
            <button
              type="button"
              onClick={() => void handleDirClick(entry)}
              className="flex w-full items-center px-4 py-2.5 border-b border-[#eaeef2] last:border-b-0 text-[#24292f] gap-3 text-sm hover:bg-[#f6f8fa] bg-transparent cursor-pointer text-left"
            >
              <span className="w-4 text-[#57606a]">📁</span>
              <span className="flex-1 font-medium">{entry.name}</span>
              <span className="text-[#57606a] text-[13px] flex-[2] overflow-hidden text-ellipsis whitespace-nowrap">
                {entry.commit_message ?? ""}
              </span>
              <span className="text-[#57606a] text-[13px] shrink-0">
                {formatRelativeTime(entry.committed_at)}
              </span>
            </button>
          ) : (
            <Link
              href={blobHref(entry, owner, repo, treeRef)}
              className="flex items-center px-4 py-2.5 border-b border-[#eaeef2] last:border-b-0 text-[#24292f] no-underline gap-3 text-sm hover:bg-[#f6f8fa]"
            >
              <span className="w-4 text-[#57606a]">📄</span>
              <span className="flex-1 font-medium">{entry.name}</span>
              <span className="text-[#57606a] text-[13px] flex-[2] overflow-hidden text-ellipsis whitespace-nowrap">
                {entry.commit_message ?? ""}
              </span>
              <span className="text-[#57606a] text-[13px] shrink-0">
                {formatRelativeTime(entry.committed_at)}
              </span>
            </Link>
          )}
          {entry.type === "dir" && loadingDirs.has(entry.path) && (
            <span className="animate-spin">…</span>
          )}
          {entry.type === "dir" && expandedDirs[entry.path] && (
            <FileTree
              entries={expandedDirs[entry.path]}
              owner={owner}
              repo={repo}
              treeRef={treeRef}
              apiClient={apiClient}
            />
          )}
        </li>
      ))}
    </ul>
  );
}
