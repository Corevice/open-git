"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

type Label = { name: string; color: string };
type Milestone = { number: number; title: string; open_issues: number };

type IssueFiltersProps = {
  owner: string;
  repo: string;
  state?: string;
  labels?: string;
  milestone?: string;
  assignee?: string;
  basePath: string;
};

function buildFilterQuery(
  basePath: string,
  overrides: {
    state?: string;
    labels?: string;
    milestone?: string;
    assignee?: string;
  },
) {
  const params = new URLSearchParams();
  const nextState = overrides.state ?? "open";

  if (nextState !== "open") params.set("state", nextState);
  if (overrides.labels) params.set("labels", overrides.labels);
  if (overrides.milestone) params.set("milestone", overrides.milestone);
  if (overrides.assignee) params.set("assignee", overrides.assignee);
  params.set("page", "1");

  return `${basePath}?${params.toString()}`;
}

export default function IssueFilters({
  owner,
  repo,
  state = "open",
  labels = "",
  milestone = "",
  assignee = "",
  basePath,
}: IssueFiltersProps) {
  const router = useRouter();
  const [allLabels, setAllLabels] = useState<Label[]>([]);
  const [milestones, setMilestones] = useState<Milestone[]>([]);

  useEffect(() => {
    let cancelled = false;

    async function loadFilters() {
      const [labelsRes, milestonesRes] = await Promise.all([
        fetch(`/repos/${owner}/${repo}/labels?per_page=100`),
        fetch(`/repos/${owner}/${repo}/milestones?state=open&per_page=100`),
      ]);

      if (cancelled) return;

      if (labelsRes.ok) setAllLabels((await labelsRes.json()) as Label[]);
      if (milestonesRes.ok) setMilestones((await milestonesRes.json()) as Milestone[]);
    }

    loadFilters();
    return () => {
      cancelled = true;
    };
  }, [owner, repo]);

  const navigate = useCallback(
    (overrides: { labels?: string; milestone?: string; assignee?: string }) => {
      const params = new URLSearchParams();
      if (state && state !== "open") params.set("state", state);
      const nextLabels = overrides.labels ?? labels;
      const nextMilestone = overrides.milestone ?? milestone;
      const nextAssignee = overrides.assignee ?? assignee;
      if (nextLabels) params.set("labels", nextLabels);
      if (nextMilestone) params.set("milestone", nextMilestone);
      if (nextAssignee) params.set("assignee", nextAssignee);
      params.set("page", "1");
      router.push(`${basePath}?${params.toString()}`);
    },
    [assignee, basePath, labels, milestone, router, state],
  );

  const currentState = state === "closed" ? "closed" : "open";

  return (
    <div className="flex flex-wrap gap-3 items-center mb-4 p-3 bg-[#f6f8fa] border border-[#d0d7de] rounded-md">
      <div className="flex gap-1">
        <Link
          href={buildFilterQuery(basePath, { state: "open", labels, milestone, assignee })}
          className={`px-3 py-1.5 text-sm rounded-md border no-underline ${
            currentState === "open"
              ? "bg-white border-[#0969da] text-[#0969da] font-semibold"
              : "bg-white border-[#d0d7de] text-[#1f2328] hover:bg-[#f6f8fa]"
          }`}
        >
          Open
        </Link>
        <Link
          href={buildFilterQuery(basePath, { state: "closed", labels, milestone, assignee })}
          className={`px-3 py-1.5 text-sm rounded-md border no-underline ${
            currentState === "closed"
              ? "bg-white border-[#0969da] text-[#0969da] font-semibold"
              : "bg-white border-[#d0d7de] text-[#1f2328] hover:bg-[#f6f8fa]"
          }`}
        >
          Closed
        </Link>
      </div>

      <select
        aria-label="Filter by label"
        value={labels.split(",").filter(Boolean)[0] ?? ""}
        onChange={(e) => navigate({ labels: e.target.value })}
        className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white"
      >
        <option value="">All labels</option>
        {allLabels.map((label) => (
          <option key={label.name} value={label.name}>
            {label.name}
          </option>
        ))}
      </select>

      <select
        aria-label="Filter by milestone"
        value={milestone}
        onChange={(e) => navigate({ milestone: e.target.value })}
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
  );
}
