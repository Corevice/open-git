"use client";

import Link from "next/link";
import { notFound } from "next/navigation";
import { use, useEffect, useState } from "react";

import { TreeBranchSelector } from "@/components/repo/BranchSelector";
import FileTree, { type TreeEntry } from "@/components/repo/FileTree";
import {
  apiClient,
  decodeBase64Content,
  decodePathSegments,
  isApiError,
} from "@/lib/api-client";
import { useRepoContents } from "@/lib/hooks/useRepoContents";
import { renderMarkdown } from "@/lib/markdown";

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
  content?: string | null;
  encoding?: string;
}

interface BranchItem {
  name: string;
}

function Skeleton({ className }: { className?: string }) {
  return (
    <div className={`animate-pulse rounded bg-[#eaeef2] ${className ?? ""}`} />
  );
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

export default function RepoTreePage({
  params,
}: {
  params: Promise<{ owner: string; repo: string; path: string[] }>;
}) {
  const { owner, repo, path: rawPathSegments } = use(params);
  const pathSegments = decodePathSegments(rawPathSegments ?? []);
  if (!pathSegments.length) notFound();

  const initialBranch = pathSegments[0];
  const currentPath = pathSegments.slice(1).join("/");

  const [currentRef, setCurrentRef] = useState(initialBranch);
  const [metadata, setMetadata] = useState<RepoMetadata | null>(null);
  const [branches, setBranches] = useState<BranchItem[]>([{ name: initialBranch }]);
  const [readmeHtml, setReadmeHtml] = useState<string | null>(null);

  const { data: contentsRaw, isLoading } = useRepoContents(
    owner,
    repo,
    currentPath,
    currentRef,
  );

  useEffect(() => {
    let cancelled = false;

    async function loadMetadata() {
      try {
        const repoData = await apiClient.getRepo<RepoMetadata>(owner, repo);
        if (cancelled) return;
        setMetadata(repoData);
      } catch (err) {
        if (isApiError(err) && err.status === 404) notFound();
        throw err;
      }
    }

    void loadMetadata();

    return () => {
      cancelled = true;
    };
  }, [owner, repo]);

  useEffect(() => {
    let cancelled = false;

    async function loadBranches() {
      try {
        const branchesRaw = await apiClient.getBranches<BranchItem[]>(owner, repo);
        if (cancelled) return;
        setBranches(
          branchesRaw.length > 0 ? branchesRaw : [{ name: currentRef }],
        );
      } catch {
        if (!cancelled) setBranches([{ name: currentRef }]);
      }
    }

    void loadBranches();

    return () => {
      cancelled = true;
    };
  }, [owner, repo, currentRef]);

  useEffect(() => {
    let cancelled = false;

    async function loadReadme() {
      if (currentPath) {
        setReadmeHtml(null);
        return;
      }

      try {
        const data = await apiClient.getContents<ContentItem>(
          owner,
          repo,
          "README.md",
          currentRef,
        );
        if (cancelled) return;
        if (!data.content || data.encoding !== "base64") {
          setReadmeHtml(null);
          return;
        }
        setReadmeHtml(renderMarkdown(decodeBase64Content(data.content)));
      } catch (err) {
        if (cancelled) return;
        if (isApiError(err) && err.status === 404) {
          setReadmeHtml(null);
          return;
        }
        throw err;
      }
    }

    void loadReadme();

    return () => {
      cancelled = true;
    };
  }, [owner, repo, currentPath, currentRef]);

  if (!metadata) {
    return (
      <div className="min-h-screen bg-[#f6f8fa] p-6">
        <div className="max-w-[1280px] mx-auto space-y-3">
          <Skeleton className="h-8 w-1/3" />
          <Skeleton className="h-5 w-full" />
          <Skeleton className="h-5 w-full" />
          <Skeleton className="h-5 w-full" />
        </div>
      </div>
    );
  }

  const contents: ContentItem[] = Array.isArray(contentsRaw)
    ? contentsRaw
    : contentsRaw
      ? [contentsRaw]
      : [];
  const entries = mapContents(contents);
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
            <TreeBranchSelector
              owner={owner}
              repo={repo}
              currentPath={currentPath}
              branches={branches}
              currentBranch={currentRef}
              onRefChange={setCurrentRef}
            />

            <nav className="flex items-center gap-1 text-sm flex-wrap min-w-0">
              <Link href={`/${owner}/${repo}`} className="text-[#0969da] hover:underline no-underline">
                {repo}
              </Link>
              <span className="text-[#57606a]">/</span>
              <Link
                href={`/${owner}/${repo}/tree/${encodeURIComponent(currentRef)}`}
                className="text-[#0969da] hover:underline no-underline font-mono"
              >
                {currentRef}
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
                        href={`/${owner}/${repo}/tree/${encodeURIComponent(currentRef)}/${sub
                          .split("/")
                          .map(encodeURIComponent)
                          .join("/")}`}
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

          {isLoading ? (
            <div className="p-4 space-y-3">
              <Skeleton className="h-5 w-full" />
              <Skeleton className="h-5 w-full" />
              <Skeleton className="h-5 w-full" />
            </div>
          ) : (
            <FileTree
              entries={entries}
              owner={owner}
              repo={repo}
              branch={currentRef}
              currentPath={currentPath}
            />
          )}
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
