"use client";

import Link from "next/link";
import { notFound, useParams, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import BranchSelector from "@/components/repo/BranchSelector";
import FileTree, { type TreeEntry } from "@/components/repo/FileTree";
import { apiClient, type ApiError } from "@/lib/api-client";

interface RepoMetadata {
  name: string;
  description: string | null;
  private: boolean;
  visibility?: string;
  default_branch: string;
  stargazers_count: number;
  watchers_count: number;
  forks_count: number;
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

type RepoApiClient = typeof apiClient & {
  getRepo(owner: string, repo: string): Promise<RepoMetadata>;
  getContents(
    owner: string,
    repo: string,
    path?: string,
    ref?: string,
  ): Promise<ContentItem[] | ContentItem>;
  getBranches(owner: string, repo: string): Promise<BranchItem[]>;
};

const client = apiClient as RepoApiClient;

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

function isNotFound(err: unknown): boolean {
  return (err as ApiError).status === 404;
}

export default function RepoTreePage() {
  const params = useParams<{ owner: string; repo: string; path: string[] }>();
  const searchParams = useSearchParams();
  const router = useRouter();

  const owner = params.owner;
  const repo = params.repo;
  const pathSegments = params.path ?? [];
  const decodedPath = pathSegments.join("/");

  const [metadata, setMetadata] = useState<RepoMetadata | null>(null);
  const [branches, setBranches] = useState<BranchItem[]>([]);
  const [entries, setEntries] = useState<TreeEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [notFoundState, setNotFoundState] = useState(false);

  const refParam = searchParams.get("ref");
  const ref = refParam ?? metadata?.default_branch ?? "main";

  useEffect(() => {
    if (!pathSegments.length) {
      setNotFoundState(true);
      return;
    }

    let cancelled = false;

    async function load() {
      setLoading(true);
      try {
        const repoData = await client.getRepo(owner, repo);
        if (cancelled) return;

        const branch = refParam ?? repoData.default_branch ?? "main";

        const [contentsRaw, branchesRaw] = await Promise.all([
          client.getContents(owner, repo, decodedPath, branch).catch((err) => {
            if (isNotFound(err)) return [] as ContentItem[];
            throw err;
          }),
          client.getBranches(owner, repo).catch(() => [{ name: branch }] as BranchItem[]),
        ]);

        if (cancelled) return;

        const contents: ContentItem[] = Array.isArray(contentsRaw)
          ? contentsRaw
          : contentsRaw
            ? [contentsRaw]
            : [];

        setMetadata(repoData);
        setBranches(branchesRaw ?? [{ name: branch }]);
        setEntries(mapContents(contents));
      } catch (err) {
        if (cancelled) return;
        if (isNotFound(err)) {
          setNotFoundState(true);
          return;
        }
        throw err;
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    load();
    return () => {
      cancelled = true;
    };
  }, [owner, repo, decodedPath, refParam, pathSegments.length]);

  if (notFoundState) {
    notFound();
  }

  if (loading || !metadata) {
    return (
      <div className="min-h-screen bg-[#f6f8fa] flex items-center justify-center text-sm text-[#57606a]">
        Loading…
      </div>
    );
  }

  const pathParts = decodedPath ? decodedPath.split("/") : [];

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
            <BranchSelector
              branches={branches}
              currentBranch={ref}
              onChange={(b) => router.push(`?ref=${encodeURIComponent(b)}`)}
            />

            <nav className="flex items-center gap-1 text-sm flex-wrap min-w-0">
              <Link href={`/${owner}/${repo}`} className="text-[#0969da] hover:underline no-underline">
                {repo}
              </Link>
              <span className="text-[#57606a]">/</span>
              <Link
                href={`/${owner}/${repo}?ref=${encodeURIComponent(ref)}`}
                className="text-[#0969da] hover:underline no-underline font-mono"
              >
                {ref}
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
                        href={`/${owner}/${repo}/tree/${sub}?ref=${encodeURIComponent(ref)}`}
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
            branch={ref}
            currentPath={decodedPath}
          />
        </div>
      </div>
    </div>
  );
}
