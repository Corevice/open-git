import Link from "next/link";

import { LabelBadge } from "@/components/issue/LabelBadge";
import { MilestoneBadge } from "@/components/issue/MilestoneBadge";

type IssueItemProps = {
  number: number;
  title: string;
  state: "open" | "closed";
  labels: Array<{ name: string; color: string }>;
  milestone?: { title: string; open_issues: number; closed_issues: number };
  assignees: Array<{ login: string }>;
  createdAt: string;
  repoPath: string;
};

export function IssueItem({
  number,
  title,
  state,
  labels,
  milestone,
  assignees,
  createdAt,
  repoPath,
}: IssueItemProps) {
  return (
    <div className="flex items-start gap-3 px-4 py-3 border-t border-[#d0d7de]">
      <span
        className={`mt-1 inline-block h-3 w-3 rounded-full ${
          state === "open" ? "bg-[#1a7f37]" : "bg-[#cf222e]"
        }`}
        aria-label={state}
      />
      <div className="flex-1 min-w-0">
        <Link
          href={`/${repoPath}/issues/${number}`}
          className="text-[15px] font-semibold text-[#1f2328] hover:text-[#0969da] no-underline"
        >
          {title}
        </Link>
        <div className="mt-1 flex flex-wrap items-center gap-1.5">
          {labels.map((label) => (
            <LabelBadge key={label.name} name={label.name} color={label.color} />
          ))}
          {milestone && (
            <MilestoneBadge
              title={milestone.title}
              openCount={milestone.open_issues}
              closedCount={milestone.closed_issues}
            />
          )}
        </div>
        <div className="mt-1 text-xs text-[#656d76]">
          #{number} opened {createdAt}
          {assignees.length > 0 && (
            <>
              {" "}
              · assigned to {assignees.map((a) => a.login).join(", ")}
            </>
          )}
        </div>
      </div>
    </div>
  );
}
