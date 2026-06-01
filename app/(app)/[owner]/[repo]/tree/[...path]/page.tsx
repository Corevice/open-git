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
}

interface ContentItem {
  name: string;
  path: string;
  type: "dir" | "file";
  sha: string;
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
  dirPath: string,
): Promise<string | null> {
  if (dirPath) return null;
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

export default async function RepoTreePage({
  params,
}: {
  params: Promise<{ owner: string; repo: string; path: string[] }>;
}) {
  const { owner, repo, path: pathSegments } = await params;
  if (!pathSegments?.length) notFound();

  const branch = pathSegments[0];
  const currentPath = pathSegments.slice(1).join("/");

  const metadata = await apiGet<RepoMetadata>(`/repos/${owner}/${repo}`);
  if (!metadata) notFound();

  const contentsPath = currentPath
    ? `/repos/${owner}/${repo}/contents/${currentPath}?ref=${encodeURIComponent(branch)}`
    : `/repos/${owner}/${repo}/contents?ref=${encodeURIComponent(branch)}`;

  const [contentsRaw, branchesRaw, readmeRaw] = await Promise.all([
    apiGet<ContentItem[] | ContentItem>(contentsPath),
    apiGet<BranchItem[]>(`/repos/${owner}/${repo}/branches?per_page=100`),
    fetchReadme(owner, repo, branch, currentPath),
  ]);

  const contents: ContentItem[] = Array.isArray(contentsRaw)
    ? contentsRaw
    : contentsRaw
      ? [contentsRaw]
      : [];
  const branches = branchesRaw ?? [{ name: branch }];
  const entries = mapContents(contents);
  const readmeHtml = readmeRaw ? renderReadmeMarkdown(readmeRaw) : null;

  const pathParts = currentPath ? currentPath.split("/") : [];

  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <header className="h-16 bg-white/85 backdrop-blur border-b border-[color:var(--border)] sticky top-0 z-[100]">
        <div className="max-w-[1280px] mx-auto px-6 flex items-center justify-between h-full">
          <Link href="/dashboard" className="text-lg font-extrabold flex items-center gap-2">
            <span>🐙</span> OpenHub
          </Link>
          <Link href="/dashboard" className="px-2 py-1 rounded-full text-xs font-medium bg-[color:var(--primary-light)] text-[color:var(--primary)]">
            {owner}
          </Link>
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
                👁 {metadata.watchers_count}
              </span>
              <span className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white inline-flex items-center gap-1.5">
                🍴 {metadata.forks_count}
              </span>
              <span className="px-3 py-1.5 text-sm bg-[color:var(--primary)] text-white rounded-md inline-flex items-center gap-1.5">
                ⭐ {metadata.stargazers_count}
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
            <Link href={`/${owner}/${repo}/issues`} className="px-4 py-2 text-sm text-[#24292f] rounded-t-md hover:bg-gray-100">
              ⊙ Issues
            </Link>
            <Link href={`/${owner}/${repo}/pulls`} className="px-4 py-2 text-sm text-[#24292f] rounded-t-md hover:bg-gray-100">
              ⇄ Pull requests
            </Link>
            <Link href={`/${owner}/${repo}/actions`} className="px-4 py-2 text-sm text-[#24292f] rounded-t-md hover:bg-gray-100">
              ▶ Actions
            </Link>
            <Link href={`/${owner}/${repo}/settings`} className="px-4 py-2 text-sm text-[#24292f] rounded-t-md hover:bg-gray-100">
              ⚙ Settings
            </Link>
          </nav>
        </div>
      </div>

      <div className="max-w-[1280px] mx-auto px-6 py-6">
        <div className="bg-white border border-[#d0d7de] rounded-lg overflow-hidden">
          <div className="p-3 border-b border-[#d0d7de] flex items-center gap-3 flex-wrap">
            <details className="relative">
              <summary className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-[#f6f8fa] border border-[#d0d7de] rounded-md text-sm cursor-pointer list-none">
                ⎇ <strong>{branch}</strong> ▾
              </summary>
              <ul className="absolute left-0 top-full mt-1 z-20 min-w-[160px] bg-white border border-[#d0d7de] rounded-md shadow-lg py-1 list-none m-0">
                {branches.map((b) => (
                  <li key={b.name}>
                    <Link
                      href={
                        currentPath
                          ? `/${owner}/${repo}/tree/${b.name}/${currentPath}`
                          : `/${owner}/${repo}?ref=${encodeURIComponent(b.name)}`
                      }
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

            <nav className="flex items-center gap-1 text-sm flex-wrap min-w-0">
              <Link href={`/${owner}/${repo}`} className="text-[#0969da] hover:underline no-underline">
                {repo}
              </Link>
              <span className="text-[#57606a]">/</span>
              <Link
                href={`/${owner}/${repo}/tree/${branch}`}
                className="text-[#0969da] hover:underline no-underline font-mono"
              >
                {branch}
              </Link>
              {pathParts.map((part, i) => {
                const sub = pathParts.slice(0, i + 1).join("/");
                const isLast = i === pathParts.length - 1;
                return (
                  <span key={sub} className="flex items-center gap-1">
                    <span className="text-[#57606a]">/</span>
                    {isLast ? (
                      <span className="font-mono text-[#24292f]">{part}</span>
                    ) : (
                      <Link
                        href={`/${owner}/${repo}/tree/${branch}/${sub}`}
                        className="text-[#0969da] hover:underline no-underline font-mono"
                      >
                        {part}
                      </Link>
                    )}
                  </span>
                );
              })}
            </nav>
          </div>

          <FileTree
            entries={entries}
            owner={owner}
            repo={repo}
            branch={branch}
            currentPath={currentPath}
          />
        </div>

        {readmeHtml && (
          <div className="bg-white border border-[#d0d7de] rounded-lg mt-6">
            <div className="p-3 border-b border-[#d0d7de] font-semibold">📖 README.md</div>
            <div
              className="p-8 prose prose-sm max-w-none"
              dangerouslySetInnerHTML={{ __html: readmeHtml }}
            />
          </div>
        )}
      </div>
    </div>
  );
}
