import Link from "next/link";
import IssueFilters from "@/components/issue/IssueFilters";
import IssueList from "@/components/issue/IssueList";

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

export default async function IssuesPage({ params, searchParams }: Props) {
  const { owner, repo } = await params;
  const { state, labels, milestone, assignee, page } = await searchParams;
  const basePath = `/${owner}/${repo}/issues`;

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <div className="max-w-[1280px] mx-auto px-6 py-6">
        <div className="flex justify-between items-center mb-4">
          <h1 className="text-2xl font-semibold">
            <span className="text-[#0969da]">{owner}</span> /{" "}
            <span className="text-[#0969da]">{repo}</span>
            <span className="ml-2 text-lg font-normal text-[#656d76]">Issues</span>
          </h1>
          <Link
            href={`/${owner}/${repo}/issues/new`}
            className="bg-[#1f883d] text-white px-4 py-1.5 rounded-md font-semibold text-sm border border-black/10 hover:bg-[#1a7f37]"
          >
            New Issue
          </Link>
        </div>

        <IssueFilters
          owner={owner}
          repo={repo}
          state={state ?? "open"}
          labels={labels}
          milestone={milestone}
          assignee={assignee}
          basePath={basePath}
        />

        <IssueList
          owner={owner}
          repo={repo}
          state={state ?? "open"}
          labels={labels}
          milestone={milestone}
          assignee={assignee}
          page={page ?? "1"}
        />
      </div>
    </div>
  );
}
