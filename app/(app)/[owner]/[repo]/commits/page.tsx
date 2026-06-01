import Link from "next/link";
import { notFound } from "next/navigation";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";

interface CommitAuthor {
  name: string;
  email: string;
  date: string;
}

interface CommitListItem {
  sha: string;
  commit: {
    message: string;
    author: CommitAuthor;
  };
  author: {
    login: string;
    avatar_url: string;
  } | null;
}

function parseLinkHeader(header: string | null): Record<string, string> {
  const links: Record<string, string> = {};
  if (!header) return links;
  for (const part of header.split(",")) {
    const match = part.match(/<([^>]+)>\s*;\s*rel="([^"]+)"/);
    if (match) links[match[2]] = match[1];
  }
  return links;
}

function pageFromUrl(url: string): number | null {
  try {
    const page = new URL(url).searchParams.get("page");
    return page ? parseInt(page, 10) : 1;
  } catch {
    return null;
  }
}

function formatRelativeTime(dateStr: string): string {
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

function authorInitials(name: string): string {
  return name
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("");
}

async function fetchCommits(
  owner: string,
  repo: string,
  page: number,
): Promise<{ commits: CommitListItem[]; links: Record<string, string> }> {
  const res = await fetch(
    `${API_BASE}/repos/${owner}/${repo}/commits?per_page=30&page=${page}`,
    {
      headers: { Accept: "application/vnd.github+json" },
      cache: "no-store",
    },
  );
  if (res.status === 404) return { commits: [], links: {} };
  if (!res.ok) throw new Error(`API commits: ${res.status}`);
  const commits = (await res.json()) as CommitListItem[];
  const links = parseLinkHeader(res.headers.get("Link"));
  return { commits, links };
}

export default async function CommitsPage({
  params,
  searchParams,
}: {
  params: Promise<{ owner: string; repo: string }>;
  searchParams: Promise<{ page?: string }>;
}) {
  const { owner, repo } = await params;
  const { page: pageParam } = await searchParams;
  const page = Math.max(1, parseInt(pageParam ?? "1", 10) || 1);

  const metadataRes = await fetch(`${API_BASE}/repos/${owner}/${repo}`, {
    headers: { Accept: "application/vnd.github+json" },
    cache: "no-store",
  });
  if (metadataRes.status === 404) notFound();
  if (!metadataRes.ok) throw new Error(`API repo: ${metadataRes.status}`);

  const { commits, links } = await fetchCommits(owner, repo, page);
  const prevPage = links.prev ? pageFromUrl(links.prev) : null;
  const nextPage = links.next ? pageFromUrl(links.next) : null;

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
          </div>

          <nav className="flex gap-1 mt-2">
            <Link
              href={`/${owner}/${repo}`}
              className="px-4 py-2 text-sm text-[#24292f] rounded-t-md hover:bg-gray-100"
            >
              Code
            </Link>
            <Link
              href={`/${owner}/${repo}/commits`}
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
                      {item.author?.avatar_url ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img
                          src={item.author.avatar_url}
                          alt={authorName}
                          className="w-8 h-8 rounded-full shrink-0"
                        />
                      ) : (
                        <span className="w-8 h-8 rounded-full shrink-0 bg-gradient-to-br from-[#54aeff] to-[#0969da] text-white text-xs font-semibold inline-flex items-center justify-center">
                          {authorInitials(item.commit.author.name)}
                        </span>
                      )}
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-semibold text-[#24292f] m-0 mb-1 truncate">
                          {message}
                        </p>
                        <p className="text-xs text-[#57606a] m-0">
                          {authorName} committed {formatRelativeTime(item.commit.author.date)}
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

          {(links.prev || links.next) && (
            <div className="px-4 py-3 border-t border-[#d0d7de] flex justify-between">
              {links.prev && prevPage ? (
                <Link
                  href={`/${owner}/${repo}/commits?page=${prevPage}`}
                  className="text-sm text-[#0969da] no-underline hover:underline"
                >
                  ← Newer
                </Link>
              ) : (
                <span />
              )}
              {links.next && nextPage ? (
                <Link
                  href={`/${owner}/${repo}/commits?page=${nextPage}`}
                  className="text-sm text-[#0969da] no-underline hover:underline"
                >
                  Older →
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
