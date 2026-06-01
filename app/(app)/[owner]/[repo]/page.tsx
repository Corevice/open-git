import Link from "next/link";
import { notFound } from "next/navigation";
import DOMPurify from "isomorphic-dompurify";
import { marked } from "marked";
import FileTree, { type TreeEntry } from "@/components/repo/FileTree";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";

interface RepoMetadata {
  name: string;
  full_name: string;
  description: string | null;
  private: boolean;
  visibility?: string;
  default_branch: string;
  stargazers_count: number;
  watchers_count: number;
  forks_count: number;
  open_issues_count?: number;
  owner: { login: string };
  html_url?: string;
}

interface ContentItem {
  name: string;
  path: string;
  type: "dir" | "file";
  sha: string;
  size?: number;
}

interface BranchItem {
  name: string;
}

async function apiGet<T>(path: string): Promise<T | null> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { Accept: "application/vnd.github+json" },
    cache: "no-store",
  });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`API ${path}: ${res.status}`);
  return res.json() as Promise<T>;
}

function visibilityLabel(repo: RepoMetadata): string {
  if (repo.visibility) {
    return repo.visibility.charAt(0).toUpperCase() + repo.visibility.slice(1);
  }
  return repo.private ? "Private" : "Public";
}

function visibilityBadgeClass(repo: RepoMetadata): string {
  const v = (repo.visibility ?? (repo.private ? "private" : "public")).toLowerCase();
  if (v === "public") return "bg-[color:var(--info-light)] text-[color:var(--info)]";
  if (v === "private") return "bg-[color:var(--warning-light)] text-[color:var(--warning)]";
  return "bg-[color:var(--info-light)] text-[color:var(--info)]";
}

function mapContents(items: ContentItem[]): TreeEntry[] {
  return items.map((item) => ({
    name: item.name,
    path: item.path,
    type: item.type === "dir" ? "dir" : "file",
    commit_message: "Update " + item.name,
  }));
}

function renderReadmeMarkdown(raw: string): string {
  const html = marked.parse(raw, { async: false }) as string;
  return DOMPurify.sanitize(html);
}

async function fetchReadme(
  owner: string,
  repo: string,
  branch: string,
): Promise<string | null> {
  const data = await apiGet<{ content?: string; encoding?: string }>(
    `/repos/${owner}/${repo}/contents/README.md?ref=${encodeURIComponent(branch)}`,
  );
  if (!data?.content || data.encoding !== "base64") return null;
  try {
    return Buffer.from(data.content, "base64").toString("utf-8");
  } catch {
    return null;
  }
}

export default async function RepoPage({
  params,
  searchParams,
}: {
  params: Promise<{ owner: string; repo: string }>;
  searchParams: Promise<{ ref?: string }>;
}) {
  const { owner, repo } = await params;
  const { ref: refParam } = await searchParams;

  const metadata = await apiGet<RepoMetadata>(`/repos/${owner}/${repo}`);
  if (!metadata) notFound();

  const branch = refParam ?? metadata.default_branch ?? "main";

  const [contentsRaw, branchesRaw, readmeRaw] = await Promise.all([
    apiGet<ContentItem[] | ContentItem>(
      `/repos/${owner}/${repo}/contents?ref=${encodeURIComponent(branch)}`,
    ),
    apiGet<BranchItem[]>(`/repos/${owner}/${repo}/branches?per_page=100`),
    fetchReadme(owner, repo, branch),
  ]);

  const contents: ContentItem[] = Array.isArray(contentsRaw)
    ? contentsRaw
    : contentsRaw
      ? [contentsRaw]
      : [];
  const branches = branchesRaw ?? [{ name: branch }];
  const entries = mapContents(contents);
  const readmeHtml = readmeRaw ? renderReadmeMarkdown(readmeRaw) : null;

  const cloneUrl = `git@${API_BASE ? "open-git.local" : "github.com"}:${owner}/${repo}.git`;

  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <header className="h-16 bg-white/85 backdrop-blur border-b border-[color:var(--border)] sticky top-0 z-[100]">
        <div className="max-w-[1280px] mx-auto px-6 flex items-center justify-between h-full">
          <Link href="/dashboard" className="text-lg font-extrabold flex items-center gap-2">
            <span>🐙</span> OpenHub
          </Link>
          <div className="flex items-center gap-3">
            <Link href="/new" className="px-2 py-1 text-sm hover:bg-gray-100 rounded">
              ＋
            </Link>
            <Link href="/dashboard" className="px-2 py-1 rounded-full text-xs font-medium bg-[color:var(--primary-light)] text-[color:var(--primary)]">
              {owner}
            </Link>
          </div>
        </div>
      </header>

      <div className="bg-white border-b border-[#d0d7de] py-4 sticky top-16 z-10">
        <div className="max-w-[1280px] mx-auto px-6">
          <div className="flex items-center gap-3 flex-wrap">
            <h1 className="text-xl m-0 flex items-center gap-2 flex-wrap">
              📁{" "}
              <Link href={`/${owner}`} className="text-[#0969da] no-underline hover:underline">
                {owner}
              </Link>
              <span>/</span>
              <Link href={`/${owner}/${repo}`} className="text-[#0969da] no-underline hover:underline">
                <strong>{repo}</strong>
              </Link>
              <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${visibilityBadgeClass(metadata)}`}>
                {visibilityLabel(metadata)}
              </span>
            </h1>
            <div className="flex gap-2 ml-auto">
              <span className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white inline-flex items-center gap-1.5">
                👁 Watch{" "}
                <span className="bg-[#eaeef2] px-2 py-0.5 rounded-full text-xs">
                  {metadata.watchers_count}
                </span>
              </span>
              <span className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white inline-flex items-center gap-1.5">
                🍴 Fork{" "}
                <span className="bg-[#eaeef2] px-2 py-0.5 rounded-full text-xs">
                  {metadata.forks_count}
                </span>
              </span>
              <span className="px-3 py-1.5 text-sm bg-[color:var(--primary)] text-white rounded-md inline-flex items-center gap-1.5">
                ⭐ Star{" "}
                <span className="bg-white/20 px-2 py-0.5 rounded-full text-xs">
                  {metadata.stargazers_count}
                </span>
              </span>
            </div>
          </div>

          <nav className="flex gap-1 mt-4">
            <Link
              href={`/${owner}/${repo}`}
              className="px-4 py-2 text-sm text-[#24292f] rounded-t-md inline-flex items-center gap-1.5 border-b-2 border-[#fd8c73] font-semibold"
            >
              📄 Code
            </Link>
            <Link
              href={`/${owner}/${repo}/issues`}
              className="px-4 py-2 text-sm text-[#24292f] rounded-t-md hover:bg-gray-100 inline-flex items-center gap-1.5"
            >
              ⊙ Issues{" "}
              {metadata.open_issues_count != null && (
                <span className="bg-[#eaeef2] px-2 py-0.5 rounded-full text-xs">
                  {metadata.open_issues_count}
                </span>
              )}
            </Link>
            <Link
              href={`/${owner}/${repo}/pulls`}
              className="px-4 py-2 text-sm text-[#24292f] rounded-t-md hover:bg-gray-100 inline-flex items-center gap-1.5"
            >
              ⇄ Pull requests
            </Link>
            <Link
              href={`/${owner}/${repo}/actions`}
              className="px-4 py-2 text-sm text-[#24292f] rounded-t-md hover:bg-gray-100 inline-flex items-center gap-1.5"
            >
              ▶ Actions
            </Link>
            <Link
              href={`/${owner}/${repo}/settings`}
              className="px-4 py-2 text-sm text-[#24292f] rounded-t-md hover:bg-gray-100 inline-flex items-center gap-1.5"
            >
              ⚙ Settings
            </Link>
          </nav>
        </div>
      </div>

      <div className="max-w-[1280px] mx-auto px-6">
        <div className="grid grid-cols-1 lg:grid-cols-[1fr_360px] gap-6 py-6">
          <div>
            <div className="bg-white border border-[#d0d7de] rounded-lg overflow-hidden">
              <div className="p-3 border-b border-[#d0d7de] flex items-center gap-3 flex-wrap">
                <details className="relative">
                  <summary className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-[#f6f8fa] border border-[#d0d7de] rounded-md text-sm text-[#24292f] hover:bg-gray-100 cursor-pointer list-none">
                    ⎇ <strong>{branch}</strong> ▾
                  </summary>
                  <ul className="absolute left-0 top-full mt-1 z-20 min-w-[160px] bg-white border border-[#d0d7de] rounded-md shadow-lg py-1 list-none m-0">
                    {branches.map((b) => (
                      <li key={b.name}>
                        <Link
                          href={`/${owner}/${repo}?ref=${encodeURIComponent(b.name)}`}
                          className={`block px-3 py-2 text-sm no-underline hover:bg-[#f6f8fa] ${
                            b.name === branch ? "font-semibold text-[#0969da]" : "text-[#24292f]"
                          }`}
                        >
                          {b.name}
                        </Link>
                      </li>
                    ))}
                  </ul>
                </details>
                <span className="text-[#57606a] text-[13px]">
                  {branches.length} branch{branches.length === 1 ? "" : "es"}
                </span>
                <div className="ml-auto flex items-center gap-2 flex-wrap">
                  <Link
                    href={`/${owner}/${repo}/commits`}
                    className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white hover:bg-gray-50"
                  >
                    ⟳ History
                  </Link>
                  <span className="bg-[#f6f8fa] border border-[#d0d7de] px-2.5 py-1.5 rounded-md font-mono text-xs max-w-[280px] truncate">
                    {cloneUrl}
                  </span>
                </div>
              </div>

              <FileTree
                entries={entries}
                owner={owner}
                repo={repo}
                branch={branch}
                currentPath=""
              />
            </div>

            {readmeHtml && (
              <div className="bg-white border border-[#d0d7de] rounded-lg mt-6">
                <div className="p-3 border-b border-[#d0d7de] font-semibold flex items-center gap-2">
                  📖 README.md
                </div>
                <div
                  className="p-8 prose prose-sm max-w-none"
                  dangerouslySetInnerHTML={{ __html: readmeHtml }}
                />
              </div>
            )}
          </div>

          <aside>
            <div className="bg-white border border-[#d0d7de] rounded-lg p-4 mb-4">
              <h3 className="text-sm m-0 mb-3 font-semibold">About</h3>
              <p className="text-[#57606a] text-sm leading-relaxed">
                {metadata.description || "No description provided."}
              </p>
              <ul className="list-none p-0 mt-3 text-[13px] text-[#57606a] space-y-1">
                <li>⭐ {metadata.stargazers_count} stars</li>
                <li>👁 {metadata.watchers_count} watching</li>
                <li>🍴 {metadata.forks_count} forks</li>
              </ul>
            </div>
          </aside>
        </div>
      </div>
    </div>
  );
}
