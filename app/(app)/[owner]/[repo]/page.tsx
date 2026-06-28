import Link from "next/link";
import { notFound } from "next/navigation";
import { RepoRefSelector } from "@/components/repo/BranchSelector";
import CloneUrlCopy from "@/components/repo/CloneUrlCopy";
import FileTree, { type TreeEntry } from "@/components/repo/FileTree";
import {
  apiClient,
  decodeBase64Content,
  isApiError,
  resolveBranchRef,
} from "@/lib/api-client";
import { renderMarkdown } from "@/lib/markdown";

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
  size?: number;
  content?: string | null;
  encoding?: string;
}

interface BranchItem {
  name: string;
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

async function fetchReadme(
  owner: string,
  repo: string,
  branch: string,
): Promise<string | null> {
  try {
    const data = await apiClient.getContents<ContentItem>(
      owner,
      repo,
      "README.md",
      branch,
    );
    if (!data.content || data.encoding !== "base64") return null;
    return decodeBase64Content(data.content);
  } catch (err) {
    if (isApiError(err) && err.status === 404) return null;
    throw err;
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

  let metadata: RepoMetadata;
  try {
    metadata = await apiClient.getRepo<RepoMetadata>(owner, repo);
  } catch (err) {
    if (isApiError(err) && err.status === 404) notFound();
    throw err;
  }

  let branchesRaw: BranchItem[];
  try {
    branchesRaw = await apiClient.getBranches<BranchItem[]>(owner, repo);
  } catch {
    branchesRaw = [{ name: metadata.default_branch ?? "main" }];
  }

  const branches = branchesRaw.length > 0 ? branchesRaw : [{ name: metadata.default_branch ?? "main" }];
  const branch = resolveBranchRef(refParam, branches, metadata.default_branch ?? "main");

  let contentsRaw: ContentItem[] | ContentItem | null = null;
  try {
    contentsRaw = await apiClient.getContents<ContentItem[] | ContentItem>(
      owner,
      repo,
      "",
      branch,
    );
  } catch (err) {
    if (!isApiError(err) || err.status !== 404) throw err;
  }

  const [readmeRaw] = await Promise.all([fetchReadme(owner, repo, branch)]);

  const contents: ContentItem[] = Array.isArray(contentsRaw)
    ? contentsRaw
    : contentsRaw
      ? [contentsRaw]
      : [];
  const entries = mapContents(contents);
  const readmeHtml = readmeRaw ? renderMarkdown(readmeRaw) : null;

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

          <div className="mt-3">
            <CloneUrlCopy owner={owner} repo={repo} />
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
            <a
              href={`/${owner}/${repo}/labels`}
              className="px-4 py-2 text-sm text-[#24292f] rounded-t-md hover:bg-gray-100 inline-flex items-center gap-1.5"
            >
              Labels
            </a>
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
                <RepoRefSelector branches={branches} currentBranch={branch} />
                <span className="text-[#57606a] text-[13px]">
                  {branches.length} branch{branches.length === 1 ? "" : "es"}
                </span>
                <div className="ml-auto flex items-center gap-2 flex-wrap">
                  <Link
                    href={`/${owner}/${repo}/commits?ref=${encodeURIComponent(branch)}`}
                    className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white hover:bg-gray-50"
                  >
                    ⟳ History
                  </Link>
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
