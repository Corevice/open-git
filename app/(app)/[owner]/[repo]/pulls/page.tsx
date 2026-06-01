"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

type Label = { name: string; color: string };
type Milestone = { number: number; title: string; open_issues: number };
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
  const router = useRouter();
  const [owner, setOwner] = useState("");
  const [repo, setRepo] = useState("");
  const [state, setState] = useState("open");
  const [labelsParam, setLabelsParam] = useState("");
  const [milestone, setMilestone] = useState("");
  const [page, setPage] = useState("1");

  const [pulls, setPulls] = useState<PullRequestItem[]>([]);
  const [allLabels, setAllLabels] = useState<Label[]>([]);
  const [milestones, setMilestones] = useState<Milestone[]>([]);
  const [pagination, setPagination] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([params, searchParams]).then(([p, sp]) => {
      setOwner(p.owner);
      setRepo(p.repo);
      setState(sp.state ?? "open");
      setLabelsParam(sp.labels ?? "");
      setMilestone(sp.milestone ?? "");
      setPage(sp.page ?? "1");
    });
  }, [params, searchParams]);

  const basePath = owner && repo ? `/${owner}/${repo}/pulls` : "";
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

  const navigate = (overrides: Record<string, string | undefined>) => {
    if (!basePath) return;
    router.push(`${basePath}${buildQuery(overrides)}`);
  };

  const toggleLabel = (name: string) => {
    const next = selectedLabels.includes(name)
      ? selectedLabels.filter((l) => l !== name)
      : [...selectedLabels, name];
    navigate({ labels: next.join(","), page: "1" });
  };

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
        if (page) query.set("page", page);
        query.set("per_page", "30");

        const [pullsRes, labelsRes, milestonesRes] = await Promise.all([
          fetch(`/repos/${owner}/${repo}/pulls?${query.toString()}`),
          fetch(`/repos/${owner}/${repo}/labels?per_page=100`),
          fetch(`/repos/${owner}/${repo}/milestones?state=open&per_page=100`),
        ]);

        if (!pullsRes.ok) throw new Error("Failed to load pull requests");

        const pullsData = (await pullsRes.json()) as PullRequestItem[];
        if (cancelled) return;

        setPulls(pullsData);
        setPagination(parseLinkHeader(pullsRes.headers.get("Link")));

        if (labelsRes.ok) setAllLabels((await labelsRes.json()) as Label[]);
        if (milestonesRes.ok) setMilestones((await milestonesRes.json()) as Milestone[]);
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
  }, [owner, repo, state, labelsParam, milestone, page]);

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

        {(pagination.prev || pagination.next) && (
          <div className="flex justify-center gap-2 py-5">
            {pagination.prev && (
              <Link
                href={linkToPath(pagination.prev).replace(/^\/repos\/[^/]+\/[^/]+\/pulls/, basePath)}
                className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm text-[#0969da]"
              >
                ← Prev
              </Link>
            )}
            {pagination.next && (
              <Link
                href={linkToPath(pagination.next).replace(/^\/repos\/[^/]+\/[^/]+\/pulls/, basePath)}
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
