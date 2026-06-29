"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";

type Label = { name: string; color: string };
type Issue = {
  number: number;
  title: string;
  state: "open" | "closed";
  user: { login: string };
  labels: Label[];
  comments: number;
  created_at: string;
};

type IssueListProps = {
  owner: string;
  repo: string;
  state?: string;
  labels?: string;
  milestone?: string;
  assignee?: string;
  page?: string;
};

function parseLinkHeader(header: string | null): Record<string, string> {
  if (!header) return {};
  const links: Record<string, string> = {};
  for (const part of header.split(",")) {
    const match = part.match(/<([^>]+)>;\s*rel="([^"]+)"/);
    if (match) links[match[2]] = match[1];
  }
  return links;
}

function formatAge(dateStr: string): string {
  const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
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

function labelStyle(color: string): React.CSSProperties {
  const hex = color.startsWith("#") ? color : `#${color}`;
  return { backgroundColor: hex, color: "#fff" };
}

export default function IssueList({
  owner,
  repo,
  state = "open",
  labels: labelsParam = "",
  milestone = "",
  assignee = "",
  page = "1",
}: IssueListProps) {
  const basePath = `/${owner}/${repo}/issues`;
  const currentPage = Math.max(1, parseInt(page, 10) || 1);

  const [issues, setIssues] = useState<Issue[]>([]);
  const [hasNext, setHasNext] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(null);
      try {
        const query = new URLSearchParams();
        if (state && state !== "all") query.set("state", state);
        if (labelsParam) query.set("labels", labelsParam);
        if (milestone) query.set("milestone", milestone);
        if (assignee) query.set("assignee", assignee);
        if (page) query.set("page", page);
        query.set("per_page", "30");

        const issuesRes = await fetch(`/repos/${owner}/${repo}/issues?${query.toString()}`);

        if (!issuesRes.ok) throw new Error("Failed to load issues");

        const issuesData = (await issuesRes.json()) as Issue[];
        if (cancelled) return;

        setIssues(issuesData);
        const pagination = parseLinkHeader(issuesRes.headers.get("Link"));
        setHasNext(!!pagination.next);
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : "Failed to load issues");
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    load();
    return () => {
      cancelled = true;
    };
  }, [owner, repo, state, labelsParam, milestone, assignee, page]);

  const openCount = issues.filter((i) => i.state === "open").length;
  const closedCount = issues.filter((i) => i.state === "closed").length;

  return (
    <div>
      <div className="flex gap-4 items-center bg-[#f6f8fa] border border-[#d0d7de] border-b-0 rounded-t-md px-4 py-3 text-sm">
        <span className="font-semibold">⊙ {openCount} Open</span>
        <span className="text-[#656d76]">✓ {closedCount} Closed</span>
      </div>

      {loading && (
        <div className="bg-white border border-[#d0d7de] px-4 py-8 text-center text-[#656d76]">
          Loading issues…
        </div>
      )}

      {error && (
        <div className="bg-white border border-[#d0d7de] px-4 py-8 text-center text-[#d1242f]">
          {error}
        </div>
      )}

      {!loading && !error && (
        <div className="bg-white border border-[#d0d7de] border-t-0 rounded-b-md">
          {issues.length === 0 ? (
            <EmptyState title="No issues" description="There are no issues matching your filters." />
          ) : (
            issues.map((issue) => (
              <div
                key={issue.number}
                className="flex items-start gap-3 px-4 py-3 border-t border-[#d0d7de] hover:bg-[#f6f8fa]"
              >
                <span className={`text-base mt-0.5 ${issue.state === "open" ? "text-[#1a7f37]" : "text-[#8250df]"}`}>
                  {issue.state === "open" ? "⊙" : "✓"}
                </span>
                <div className="flex-1 min-w-0">
                  <Link
                    href={`/${owner}/${repo}/issues/${issue.number}`}
                    className="text-[15px] font-semibold text-[#1f2328] hover:text-[#0969da] no-underline"
                  >
                    {issue.title}
                  </Link>
                  {issue.labels.map((label) => (
                    <span
                      key={label.name}
                      className="inline-block px-2 ml-1.5 rounded-[10px] text-[11px] font-semibold leading-[18px]"
                      style={labelStyle(label.color)}
                    >
                      {label.name}
                    </span>
                  ))}
                  <div className="text-xs text-[#656d76] mt-1">
                    #{issue.number} opened {formatAge(issue.created_at)} by {issue.user.login}
                  </div>
                </div>
                <div className="text-xs text-[#656d76] whitespace-nowrap">💬 {issue.comments}</div>
              </div>
            ))
          )}
        </div>
      )}

      {!loading && !error && (
        <Pagination
          page={currentPage}
          hasNext={hasNext}
          hasPrev={currentPage > 1}
          basePath={basePath}
        />
      )}
    </div>
  );
}
