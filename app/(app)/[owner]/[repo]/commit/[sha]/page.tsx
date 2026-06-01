import Link from "next/link";
import { notFound } from "next/navigation";
import DiffViewer from "@/components/repo/DiffViewer";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";

interface CommitFile {
  filename: string;
  status: string;
  additions: number;
  deletions: number;
  changes: number;
  patch?: string;
}

interface CommitDetail {
  sha: string;
  commit: {
    message: string;
    author: {
      name: string;
      email: string;
      date: string;
    };
  };
  author: {
    login: string;
    avatar_url: string;
  } | null;
  stats?: {
    total: number;
    additions: number;
    deletions: number;
  };
  files?: CommitFile[];
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

async function fetchCommit(
  owner: string,
  repo: string,
  sha: string,
): Promise<CommitDetail | null> {
  const res = await fetch(`${API_BASE}/repos/${owner}/${repo}/commits/${sha}`, {
    headers: { Accept: "application/vnd.github+json" },
    cache: "no-store",
  });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`API commit: ${res.status}`);
  return res.json() as Promise<CommitDetail>;
}

export default async function CommitPage({
  params,
}: {
  params: Promise<{ owner: string; repo: string; sha: string }>;
}) {
  const { owner, repo, sha } = await params;

  const commit = await fetchCommit(owner, repo, sha);
  if (!commit) notFound();

  const files = commit.files ?? [];
  const additions = commit.stats?.additions ?? files.reduce((sum, f) => sum + f.additions, 0);
  const deletions = commit.stats?.deletions ?? files.reduce((sum, f) => sum + f.deletions, 0);
  const authorName = commit.author?.login ?? commit.commit.author.name;
  const message = commit.commit.message.split("\n")[0];

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

      <div className="max-w-[1280px] mx-auto px-6 py-6">
        <div className="text-sm text-[#57606a] mb-4">
          <Link href={`/${owner}`} className="text-[#0969da] no-underline hover:underline">
            {owner}
          </Link>
          <span> / </span>
          <Link href={`/${owner}/${repo}`} className="text-[#0969da] no-underline hover:underline">
            {repo}
          </Link>
          <span> / </span>
          <Link href={`/${owner}/${repo}/commits`} className="text-[#0969da] no-underline hover:underline">
            Commits
          </Link>
          <span> / </span>
          <code className="font-mono text-[#24292f]">{sha.slice(0, 7)}</code>
        </div>

        <div className="bg-white border border-[#d0d7de] rounded-lg overflow-hidden mb-6">
          <div className="px-6 py-5 border-b border-[#d0d7de]">
            <div className="flex items-start gap-3">
              {commit.author?.avatar_url ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={commit.author.avatar_url}
                  alt={authorName}
                  className="w-10 h-10 rounded-full shrink-0"
                />
              ) : (
                <span className="w-10 h-10 rounded-full shrink-0 bg-gradient-to-br from-[#54aeff] to-[#0969da] text-white text-sm font-semibold inline-flex items-center justify-center">
                  {authorInitials(commit.commit.author.name)}
                </span>
              )}
              <div>
                <h1 className="text-lg font-semibold text-[#24292f] m-0 mb-1">{message}</h1>
                <p className="text-sm text-[#57606a] m-0">
                  <strong className="text-[#24292f]">{authorName}</strong> committed{" "}
                  {formatRelativeTime(commit.commit.author.date)}
                </p>
                <code className="inline-block mt-2 text-xs font-mono bg-[#f6f8fa] text-[#0969da] px-2 py-1 rounded">
                  {commit.sha}
                </code>
              </div>
            </div>
          </div>

          <div className="px-6 py-3 bg-[#f6f8fa] border-b border-[#d0d7de] text-sm text-[#57606a] flex flex-wrap gap-4">
            <span>
              <strong className="text-[#24292f]">{files.length}</strong>{" "}
              {files.length === 1 ? "file" : "files"} changed
            </span>
            {additions > 0 && (
              <span className="text-green-700 font-semibold">+{additions} insertions</span>
            )}
            {deletions > 0 && (
              <span className="text-red-700 font-semibold">-{deletions} deletions</span>
            )}
          </div>
        </div>

        {files.length === 0 ? (
          <p className="text-sm text-[#57606a]">No file changes in this commit.</p>
        ) : (
          files.map((file) => (
            <DiffViewer
              key={file.filename}
              filename={file.filename}
              patch={file.patch}
              additions={file.additions}
              deletions={file.deletions}
            />
          ))
        )}
      </div>
    </div>
  );
}
