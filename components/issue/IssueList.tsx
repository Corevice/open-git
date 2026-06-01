"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

type Label = { name: string; color: string };
type Milestone = { number: number; title: string; open_issues: number };
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

function linkToPath(url: string): string {
  try {
    const parsed = new URL(url, "http://localhost");
    return `${parsed.pathname}${parsed.search}`;
  } catch {
    return url;
  }
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
  page = "1",
}: IssueListProps) {
  const router = useRouter();
  const basePath = `/${owner}/${repo}/issues`;

  const [issues, setIssues] = useState<Issue[]>([]);
  const [allLabels, setAllLabels] = useState<Label[]>([]);
  const [milestones, setMilestones] = useState<Milestone[]>([]);
  const [pagination, setPagination] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const selectedLabels = labelsParam ? labelsParam.split(",").filter(Boolean) : [];

  const buildQuery = useCallback(
    (overrides: Record<string, string | undefined> = {}) => {
      const params = new URLSearchParams();
      const nextState = overrides.state ?? state;
      const nextLabels = overrides.labels ?? labelsParam;
      const nextMilestone = overrides.milestone ?? milestone;
      const nextPage = overrides.page ?? page;

      if (nextState && nextState !== "all") params.set("state", nextState);
      if (nextLabels) params.set("labels", nextLabels);
      if (nextMilestone) params.set("milestone", nextMilestone);
      if (nextPage && nextPage !== "1") params.set("page", nextPage);

      const qs = params.toString();
      return qs ? `?${qs}` : "";
    },
    [state, labelsParam, milestone, page],
  );

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
        if (page) query.set("page", page);
        query.set("per_page", "30");

        const [issuesRes, labelsRes, milestonesRes] = await Promise.all([
          fetch(`/repos/${owner}/${repo}/issues?${query.toString()}`),
          fetch(`/repos/${owner}/${repo}/labels?per_page=100`),
          fetch(`/repos/${owner}/${repo}/milestones?state=open&per_page=100`),
        ]);

        if (!issuesRes.ok) throw new Error("Failed to load issues");

        const issuesData = (await issuesRes.json()) as Issue[];
        if (cancelled) return;

        setIssues(issuesData);
        setPagination(parseLinkHeader(issuesRes.headers.get("Link")));

        if (labelsRes.ok) setAllLabels((await labelsRes.json()) as Label[]);
        if (milestonesRes.ok) setMilestones((await milestonesRes.json()) as Milestone[]);
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
  }, [owner, repo, state, labelsParam, milestone, page]);

  const navigate = (overrides: Record<string, string | undefined>) => {
    router.push(`${basePath}${buildQuery(overrides)}`);
  };

  const toggleLabel = (name: string) => {
    const next = selectedLabels.includes(name)
      ? selectedLabels.filter((l) => l !== name)
      : [...selectedLabels, name];
    navigate({ labels: next.join(","), page: "1" });
  };

  const openCount = issues.filter((i) => i.state === "open").length;
  const closedCount = issues.filter((i) => i.state === "closed").length;

  return (
    <div>
      <div className="flex flex-wrap gap-3 items-center mb-4 p-3 bg-[#f6f8fa] border border-[#d0d7de] rounded-md">
        <div className="flex gap-1">
          {(["open", "closed", "all"] as const).map((s) => (
            <button
              key={s}
              type="button"
              onClick={() => navigate({ state: s, page: "1" })}
              className={`px-3 py-1.5 text-sm rounded-md border ${
                state === s
                  ? "bg-white border-[#0969da] text-[#0969da] font-semibold"
                  : "bg-white border-[#d0d7de] hover:bg-[#f6f8fa]"
              }`}
            >
              {s === "open" ? "Open" : s === "closed" ? "Closed" : "All"}
            </button>
          ))}
        </div>

        <div className="flex flex-wrap gap-1.5 items-center">
          <span className="text-xs text-[#656d76] font-semibold uppercase">Labels:</span>
          {allLabels.map((label) => {
            const selected = selectedLabels.includes(label.name);
            return (
              <button
                key={label.name}
                type="button"
                onClick={() => toggleLabel(label.name)}
                className={`px-2 py-0.5 rounded-full text-[11px] font-semibold border ${
                  selected ? "ring-2 ring-[#0969da] ring-offset-1" : "opacity-80 hover:opacity-100"
                }`}
                style={labelStyle(label.color)}
              >
                {label.name}
              </button>
            );
          })}
        </div>

        <select
          value={milestone}
          onChange={(e) => navigate({ milestone: e.target.value, page: "1" })}
          className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white"
        >
          <option value="">All milestones</option>
          {milestones.map((m) => (
            <option key={m.number} value={String(m.number)}>
              {m.title} ({m.open_issues} open)
            </option>
          ))}
        </select>
      </div>

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
            <div className="px-4 py-8 text-center text-[#656d76]">No issues found.</div>
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

      {(pagination.prev || pagination.next) && (
        <div className="flex justify-center gap-2 py-5">
          {pagination.prev && (
            <Link
              href={linkToPath(pagination.prev).replace(/^\/repos\/[^/]+\/[^/]+\/issues/, basePath)}
              className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm text-[#0969da]"
            >
              ← Prev
            </Link>
          )}
          {pagination.next && (
            <Link
              href={linkToPath(pagination.next).replace(/^\/repos\/[^/]+\/[^/]+\/issues/, basePath)}
              className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm text-[#0969da]"
            >
              Next →
            </Link>
          )}
        </div>
      )}
    </div>
  );
}
