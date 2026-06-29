"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

import { ApiError, listPullRequests, listPullRequestsWithPagination } from "@/lib/api";
import type { PullRequest } from "@/types/pull-request";

type Tab = "open" | "closed" | "merged";

type Props = {
  params: Promise<{ owner: string; repo: string }>;
  searchParams: Promise<{ state?: string; page?: string }>;
};

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

function tabFromState(state: string | undefined): Tab {
  if (state === "closed") return "closed";
  if (state === "merged") return "merged";
  return "open";
}

function stateFromTab(tab: Tab): string {
  if (tab === "merged") return "closed";
  return tab;
}

function filterByTab(items: PullRequest[], tab: Tab): PullRequest[] {
  if (tab === "open") {
    return items.filter((pr) => pr.state === "open" && !pr.merged_at);
  }
  if (tab === "merged") {
    return items.filter((pr) => pr.merged_at != null);
  }
  return items.filter((pr) => pr.state === "closed" && !pr.merged_at);
}

function hasConflicts(pr: PullRequest): boolean {
  return (
    pr.mergeable === false ||
    pr.mergeable_state === "dirty" ||
    pr.mergeable_state === "conflicting"
  );
}

function prStatusBadge(pr: PullRequest): {
  label: string;
  className: string;
  iconClass: string;
} {
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

function paginationHref(linkUrl: string, basePath: string): string {
  try {
    const url = new URL(linkUrl, "http://localhost");
    const params = new URLSearchParams();
    const state = url.searchParams.get("state");
    const pageParam = url.searchParams.get("page");
    if (state && state !== "open") params.set("state", state);
    if (pageParam && pageParam !== "1") params.set("page", pageParam);
    const query = params.toString();
    return query ? `${basePath}?${query}` : basePath;
  } catch {
    return basePath;
  }
}

function authorLogin(pr: PullRequest): string {
  return pr.user?.login ?? pr.author_id ?? "unknown";
}

function LoadingSkeleton() {
  return (
    <div className="bg-white border border-[#d0d7de] border-t-0 rounded-b-md">
      {Array.from({ length: 5 }).map((_, i) => (
        <div
          key={i}
          className="flex items-start gap-3 px-4 py-3 border-t border-[#d0d7de] animate-pulse"
        >
          <div className="w-4 h-4 bg-[#eaeef2] rounded mt-1" />
          <div className="flex-1 space-y-2">
            <div className="h-4 bg-[#eaeef2] rounded w-2/3" />
            <div className="h-3 bg-[#eaeef2] rounded w-1/3" />
          </div>
        </div>
      ))}
    </div>
  );
}

export default function PullsPage({ params, searchParams }: Props) {
  const router = useRouter();
  const [owner, setOwner] = useState("");
  const [repo, setRepo] = useState("");
  const [activeTab, setActiveTab] = useState<Tab>("open");
  const [page, setPage] = useState("1");

  const [pulls, setPulls] = useState<PullRequest[]>([]);
  const [pagination, setPagination] = useState<Record<string, string>>({});
  const [tabCounts, setTabCounts] = useState({ open: 0, closed: 0, merged: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([params, searchParams]).then(([p, sp]) => {
      setOwner(p.owner);
      setRepo(p.repo);
      setActiveTab(tabFromState(sp.state));
      setPage(sp.page ?? "1");
    });
  }, [params, searchParams]);

  const basePath = owner && repo ? `/${owner}/${repo}/pulls` : "";

  const buildQuery = useCallback(
    (overrides: { state?: Tab; page?: string } = {}) => {
      const qs = new URLSearchParams();
      const nextTab = overrides.state ?? activeTab;
      const nextPage = overrides.page ?? page;

      if (nextTab !== "open") qs.set("state", nextTab);
      if (nextPage && nextPage !== "1") qs.set("page", nextPage);

      const query = qs.toString();
      return query ? `?${query}` : "";
    },
    [activeTab, page],
  );

  const navigate = (overrides: { state?: Tab; page?: string }) => {
    if (!basePath) return;
    router.push(`${basePath}${buildQuery(overrides)}`);
  };

  useEffect(() => {
    if (!owner || !repo) return;

    let cancelled = false;

    async function loadCounts() {
      try {
        const [openData, closedData] = await Promise.all([
          listPullRequests(owner, repo, "open", 1, 100),
          listPullRequests(owner, repo, "closed", 1, 100),
        ]);
        if (cancelled) return;
        setTabCounts({
          open: filterByTab(openData, "open").length,
          closed: filterByTab(closedData, "closed").length,
          merged: filterByTab(closedData, "merged").length,
        });
      } catch {
        // tab counts are best-effort
      }
    }

    loadCounts();
    return () => {
      cancelled = true;
    };
  }, [owner, repo]);

  useEffect(() => {
    if (!owner || !repo) return;

    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(null);
      try {
        const pageNum = parseInt(page, 10) || 1;
        const { items, links } = await listPullRequestsWithPagination(
          owner,
          repo,
          stateFromTab(activeTab),
          pageNum,
          30,
        );
        if (cancelled) return;
        setPulls(filterByTab(items, activeTab));
        setPagination(links);
      } catch (e) {
        if (cancelled) return;
        if (e instanceof ApiError && e.status === 404) {
          router.push("/404");
          return;
        }
        setError(e instanceof Error ? e.message : "Failed to load pull requests");
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    load();
    return () => {
      cancelled = true;
    };
  }, [owner, repo, activeTab, page, router]);

  const openCount = tabCounts.open;
  const closedCount = tabCounts.closed;
  const mergedCount = tabCounts.merged;

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

        <div className="flex flex-wrap gap-3 items-center mb-4 p-3 bg-[#f6f8fa] border border-[#d0d7de] rounded-md">
          <div className="flex gap-1">
            {(
              [
                { id: "open" as const, label: "Open" },
                { id: "closed" as const, label: "Closed" },
                { id: "merged" as const, label: "Merged" },
              ] as const
            ).map(({ id, label }) => (
              <button
                key={id}
                type="button"
                onClick={() => navigate({ state: id, page: "1" })}
                className={`px-3 py-1.5 text-sm rounded-md border ${
                  activeTab === id
                    ? "bg-white border-[#0969da] text-[#0969da] font-semibold"
                    : "bg-white border-[#d0d7de] hover:bg-[#f6f8fa]"
                }`}
              >
                {label}
              </button>
            ))}
          </div>
        </div>

        <div className="flex gap-4 items-center bg-[#f6f8fa] border border-[#d0d7de] border-b-0 rounded-t-md px-4 py-3 text-sm">
          <span className="font-semibold text-[#1a7f37]">⇆ {openCount} Open</span>
          <span className="text-[#656d76]">✓ {closedCount} Closed</span>
          <span className="text-[#656d76]">⊕ {mergedCount} Merged</span>
        </div>

        {loading && <LoadingSkeleton />}

        {error && (
          <div className="bg-white border border-[#d0d7de] px-4 py-8 text-center text-[#d1242f]">
            {error}
          </div>
        )}

        {!loading && !error && (
          <div className="bg-white border border-[#d0d7de] border-t-0 rounded-b-md">
            {pulls.length === 0 ? (
              <div className="px-4 py-8 text-center text-[#656d76]">No pull requests found.</div>
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
                      {pr.labels?.map((label) => (
                        <span
                          key={label.name}
                          className="inline-block ml-1 px-2 py-0.5 rounded-full text-[11px] font-semibold"
                          style={labelStyle(label.color)}
                        >
                          {label.name}
                        </span>
                      ))}
                      <div className="text-xs text-[#656d76] mt-1">
                        #{pr.number} opened {formatAge(pr.created_at)} by {authorLogin(pr)}
                      </div>
                    </div>
                  </div>
                );
              })
            )}
          </div>
        )}

        {(pagination.prev || pagination.next) && (
          <div className="flex justify-center gap-2 py-5">
            {pagination.prev && (
              <Link
                href={paginationHref(pagination.prev, basePath)}
                className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm text-[#0969da]"
              >
                ← Prev
              </Link>
            )}
            {pagination.next && (
              <Link
                href={paginationHref(pagination.next, basePath)}
                className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm text-[#0969da]"
              >
                Next →
              </Link>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
