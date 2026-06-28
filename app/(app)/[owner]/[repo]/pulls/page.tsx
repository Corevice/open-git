"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import IssueFilters from "@/components/issue/IssueFilters";
import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";

type PullRequestItem = {
  number: number;
  title: string;
  state: "open" | "closed";
  draft?: boolean;
  merged_at?: string | null;
  mergeable_state?: string;
  user: { login: string };
  created_at: string;
  comments?: number;
};

type Props = {
  params: Promise<{ owner: string; repo: string }>;
  searchParams: Promise<{
    state?: string;
    labels?: string;
    milestone?: string;
    assignee?: string;
    page?: string;
  }>;
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

function hasConflicts(pr: PullRequestItem): boolean {
  return pr.mergeable_state === "dirty" || pr.mergeable_state === "conflicting";
}

function prStatusBadge(pr: PullRequestItem): { label: string; className: string; iconClass: string } {
  if (pr.merged_at) {
    return {
      label: "Merged",
      className: "bg-[#8250df] text-white",
      iconClass: "text-[#8250df]",
    };
  }
  if (pr.draft) {
    return {
      label: "Draft",
      className: "bg-[#eaeef2] text-[#57606a]",
      iconClass: "text-[#6e7781]",
    };
  }
  if (hasConflicts(pr)) {
    return {
      label: "Conflicts",
      className: "bg-[#ffebe9] text-[#cf222e]",
      iconClass: "text-[#cf222e]",
    };
  }
  if (pr.state === "open") {
    return {
      label: "Open",
      className: "bg-[#dafbe1] text-[#1a7f37]",
      iconClass: "text-[#1f883d]",
    };
  }
  return {
    label: "Closed",
    className: "bg-[#8250df] text-white",
    iconClass: "text-[#8250df]",
  };
}

export default function PullsPage({ params, searchParams }: Props) {
  const [owner, setOwner] = useState("");
  const [repo, setRepo] = useState("");
  const [state, setState] = useState("open");
  const [labelsParam, setLabelsParam] = useState("");
  const [milestone, setMilestone] = useState("");
  const [assignee, setAssignee] = useState("");
  const [page, setPage] = useState("1");

  const [pulls, setPulls] = useState<PullRequestItem[]>([]);
  const [hasNext, setHasNext] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([params, searchParams]).then(([p, sp]) => {
      setOwner(p.owner);
      setRepo(p.repo);
      setState(sp.state ?? "open");
      setLabelsParam(sp.labels ?? "");
      setMilestone(sp.milestone ?? "");
      setAssignee(sp.assignee ?? "");
      setPage(sp.page ?? "1");
    });
  }, [params, searchParams]);

  const basePath = owner && repo ? `/${owner}/${repo}/pulls` : "";
  const currentPage = Math.max(1, parseInt(page, 10) || 1);

  useEffect(() => {
    if (!owner || !repo) return;

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

        const pullsRes = await fetch(`/repos/${owner}/${repo}/pulls?${query.toString()}`);

        if (!pullsRes.ok) throw new Error("Failed to load pull requests");

        const pullsData = (await pullsRes.json()) as PullRequestItem[];
        if (cancelled) return;

        setPulls(pullsData);
        const pagination = parseLinkHeader(pullsRes.headers.get("Link"));
        setHasNext(!!pagination.next);
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : "Failed to load pull requests");
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    load();
    return () => {
      cancelled = true;
    };
  }, [owner, repo, state, labelsParam, milestone, assignee, page]);

  const openCount = pulls.filter((p) => p.state === "open" && !p.merged_at).length;
  const closedCount = pulls.filter((p) => p.state === "closed" || p.merged_at).length;

  if (!owner || !repo) {
    return (
      <div className="min-h-screen bg-[#f6f8fa] flex items-center justify-center text-[#656d76]">
        Loading…
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <div className="max-w-[1280px] mx-auto px-6 py-6">
        <div className="flex justify-between items-center mb-4">
          <h1 className="text-2xl font-semibold">
            <span className="text-[#0969da]">{owner}</span> /{" "}
            <span className="text-[#0969da]">{repo}</span>
            <span className="ml-2 text-lg font-normal text-[#656d76]">Pull Requests</span>
          </h1>
          <Link
            href={`/${owner}/${repo}/pulls/new`}
            className="bg-[#1f883d] text-white px-4 py-1.5 rounded-md font-semibold text-sm border border-black/10 hover:bg-[#1a7f37]"
          >
            New Pull Request
          </Link>
        </div>

        <IssueFilters
          owner={owner}
          repo={repo}
          state={state}
          labels={labelsParam}
          milestone={milestone}
          assignee={assignee}
          basePath={basePath}
        />

        <div className="flex gap-4 items-center bg-[#f6f8fa] border border-[#d0d7de] border-b-0 rounded-t-md px-4 py-3 text-sm">
          <span className="font-semibold text-[#1a7f37]">⇆ {openCount} Open</span>
          <span className="text-[#656d76]">✓ {closedCount} Closed</span>
        </div>

        {loading && (
          <div className="bg-white border border-[#d0d7de] px-4 py-8 text-center text-[#656d76]">
            Loading pull requests…
          </div>
        )}

        {error && (
          <div className="bg-white border border-[#d0d7de] px-4 py-8 text-center text-[#d1242f]">
            {error}
          </div>
        )}

        {!loading && !error && (
          <div className="bg-white border border-[#d0d7de] border-t-0 rounded-b-md">
            {pulls.length === 0 ? (
              <EmptyState
                title="No pull requests"
                description="There are no pull requests matching your filters."
              />
            ) : (
              pulls.map((pr) => {
                const badge = prStatusBadge(pr);
                return (
                  <div
                    key={pr.number}
                    className="flex items-start gap-3 px-4 py-3 border-t border-[#d0d7de] hover:bg-[#f6f8fa]"
                  >
                    <span className={`text-base mt-0.5 ${badge.iconClass}`}>⇆</span>
                    <div className="flex-1 min-w-0">
                      <Link
                        href={`/${owner}/${repo}/pull/${pr.number}`}
                        className="text-[15px] font-semibold text-[#1f2328] hover:text-[#0969da] no-underline"
                      >
                        {pr.title}
                      </Link>
                      <span
                        className={`inline-block ml-2 px-2 py-0.5 rounded-full text-[11px] font-semibold ${badge.className}`}
                      >
                        {badge.label}
                      </span>
                      <div className="text-xs text-[#656d76] mt-1">
                        #{pr.number} opened {formatAge(pr.created_at)} by {pr.user.login}
                      </div>
                    </div>
                    {pr.comments != null && (
                      <div className="text-xs text-[#656d76] whitespace-nowrap">💬 {pr.comments}</div>
                    )}
                  </div>
                );
              })
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
    </div>
  );
}
