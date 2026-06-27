import Link from "next/link";
import { notFound } from "next/navigation";
import { apiClient, type ApiError } from "@/lib/api-client";

interface CommitAuthor {
  name: string;
  email?: string;
  date: string;
}

interface CommitListItem {
  sha: string;
  commit: {
    message: string;
    author: CommitAuthor;
  };
  author?: {
    login: string;
    avatar_url?: string;
  } | null;
}

type RepoApiClient = typeof apiClient & {
  getRepo(owner: string, repo: string): Promise<{ default_branch: string }>;
  getCommits(
    owner: string,
    repo: string,
    branch: string,
    page: number,
  ): Promise<CommitListItem[]>;
};

const client = apiClient as RepoApiClient;

const PER_PAGE = 30;

function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) return dateStr;
  return date.toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function isNotFound(err: unknown): boolean {
  return (err as ApiError).status === 404;
}

export default async function CommitsPage({
  params,
  searchParams,
}: {
  params: Promise<{ owner: string; repo: string }>;
  searchParams: Promise<{ page?: string; ref?: string }>;
}) {
  const { owner, repo } = await params;
  const { page: pageParam, ref: refParam } = await searchParams;
  const page = Math.max(1, parseInt(pageParam ?? "1", 10) || 1);

  let repoData: { default_branch: string };
  try {
    repoData = await client.getRepo(owner, repo);
  } catch (err) {
    if (isNotFound(err)) notFound();
    throw err;
  }

  const branch = refParam ?? repoData.default_branch ?? "main";

  let commits: CommitListItem[];
  try {
    commits = await client.getCommits(owner, repo, branch, page);
  } catch (err) {
    if (isNotFound(err)) {
      commits = [];
    } else {
      throw err;
    }
  }

  const hasPrev = page > 1;
  const hasNext = commits.length === PER_PAGE;

  const buildPageHref = (targetPage: number) => {
    const params = new URLSearchParams();
    params.set("page", String(targetPage));
    if (refParam) {
      params.set("ref", branch);
    }
    return `/${owner}/${repo}/commits?${params.toString()}`;
  };

  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <header className="h-16 bg-white/85 backdrop-blur border-b border-[color:var(--border)] sticky top-0 z-[100]">
        <div className="max-w-[1280px] mx-auto px-6 flex items-center justify-between h-full">
          <Link href="/dashboard" className="text-lg font-extrabold flex items-center gap-2">
            <span>🐙</span> OpenHub
          </Link>
          <Link
            href="/dashboard"
            className="px-2 py-1 rounded-full text-xs font-medium bg-[color:var(--primary-light)] text-[color:var(--primary)]"
          >
            {owner}
          </Link>
        </div>
      </header>

      <div className="bg-white border-b border-[#d0d7de] py-4">
        <div className="max-w-[1280px] mx-auto px-6">
          <div className="text-sm text-[#57606a] mb-2">
            <Link href={`/${owner}`} className="text-[#0969da] no-underline hover:underline">
              {owner}
            </Link>
            <span> / </span>
            <Link href={`/${owner}/${repo}`} className="text-[#0969da] no-underline hover:underline">
              {repo}
            </Link>
            <span> / </span>
            <strong className="text-[#24292f]">Commits</strong>
            <span className="text-[#57606a]"> ({branch})</span>
          </div>

          <nav className="flex gap-1 mt-2">
            <Link
              href={`/${owner}/${repo}`}
              className="px-4 py-2 text-sm text-[#24292f] rounded-t-md hover:bg-gray-100"
            >
              Code
            </Link>
            <Link
              href={`/${owner}/${repo}/commits${refParam ? `?ref=${encodeURIComponent(branch)}` : ""}`}
              className="px-4 py-2 text-sm text-[#24292f] rounded-t-md border-b-2 border-[#fd8c73] font-semibold"
            >
              Commits
            </Link>
          </nav>
        </div>
      </div>

      <div className="max-w-[1280px] mx-auto px-6 py-6">
        <div className="bg-white border border-[#d0d7de] rounded-lg overflow-hidden">
          <div className="px-4 py-3 border-b border-[#d0d7de] font-semibold text-sm">
            Commit history
          </div>

          {commits.length === 0 ? (
            <p className="px-4 py-8 text-sm text-[#57606a]">No commits found.</p>
          ) : (
            <ul className="list-none p-0 m-0">
              {commits.map((item) => {
                const message = item.commit.message.split("\n")[0];
                const authorName = item.author?.login ?? item.commit.author.name;
                return (
                  <li key={item.sha} className="border-b border-[#d8dee4] last:border-b-0">
                    <Link
                      href={`/${owner}/${repo}/commit/${item.sha}`}
                      className="flex items-start gap-3 px-4 py-4 no-underline hover:bg-[#f6f8fa]"
                    >
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-semibold text-[#24292f] m-0 mb-1 truncate">
                          {message}
                        </p>
                        <p className="text-xs text-[#57606a] m-0">
                          {authorName} committed {formatDate(item.commit.author.date)}
                        </p>
                      </div>
                      <code className="text-xs font-mono bg-[#f6f8fa] text-[#0969da] px-2 py-1 rounded shrink-0">
                        {item.sha.slice(0, 7)}
                      </code>
                    </Link>
                  </li>
                );
              })}
            </ul>
          )}

          {(hasPrev || hasNext) && (
            <div className="px-4 py-3 border-t border-[#d0d7de] flex justify-between">
              {hasPrev ? (
                <Link
                  href={buildPageHref(page - 1)}
                  className="text-sm text-[#0969da] no-underline hover:underline"
                >
                  ← Previous
                </Link>
              ) : (
                <span />
              )}
              {hasNext ? (
                <Link
                  href={buildPageHref(page + 1)}
                  className="text-sm text-[#0969da] no-underline hover:underline"
                >
                  Next →
                </Link>
              ) : (
                <span />
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
