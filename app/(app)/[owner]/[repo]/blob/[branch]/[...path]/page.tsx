"use client";

import Link from "next/link";
import { notFound } from "next/navigation";
import { use, useEffect, useState } from "react";

import { RepoRefSelector } from "@/components/repo/BranchSelector";
import BlobViewer from "@/components/repo/BlobViewer";
import {
  apiClient,
  decodeBase64Content,
  decodePathSegments,
} from "@/lib/api-client";
import { useRepoContents } from "@/lib/hooks/useRepoContents";
import { RepoPageSkeleton } from "@/app/(app)/[owner]/[repo]/tree/[...path]/page";

interface BranchItem {
  name: string;
}

function isBinaryContent(data: { content?: string | null }): boolean {
  return data.content == null;
}

export default function BlobPage({
  params,
}: {
  params: Promise<{ owner: string; repo: string; branch: string; path: string[] }>;
}) {
  const { owner, repo, branch: rawBranch, path: rawPathSegments } = use(params);
  const initialBranch = decodeURIComponent(rawBranch);
  const decodedPath = decodePathSegments(rawPathSegments ?? []).join("/");

  const [currentRef, setCurrentRef] = useState(initialBranch);
  const [branches, setBranches] = useState<BranchItem[]>([{ name: initialBranch }]);

  const {
    data: contentData,
    isLoading,
    error,
    isNotFound,
  } = useRepoContents(owner, repo, decodedPath, currentRef);

  useEffect(() => {
    let cancelled = false;

    async function loadBranches() {
      try {
        const branchesRaw = await apiClient.getBranches<BranchItem[]>(owner, repo);
        if (cancelled) return;
        setBranches(
          branchesRaw.length > 0 ? branchesRaw : [{ name: initialBranch }],
        );
      } catch {
        if (!cancelled) setBranches([{ name: initialBranch }]);
      }
    }

    void loadBranches();

    return () => {
      cancelled = true;
    };
  }, [owner, repo, initialBranch]);

  if (isLoading) {
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
          <div className="bg-white border border-[#d0d7de] rounded-lg overflow-hidden p-4 space-y-3">
            <RepoPageSkeleton className="h-5 w-full" />
            <RepoPageSkeleton className="h-5 w-full" />
            <RepoPageSkeleton className="h-5 w-full" />
          </div>
        </div>
      </div>
    );
  }

  if (isNotFound) notFound();
  if (error) throw error;
  if (!contentData || Array.isArray(contentData)) notFound();
  if (contentData.type !== "file") notFound();

  const fileData = contentData;
  const downloadUrl = fileData.download_url ?? "";
  const pathParts = decodedPath.split("/");
  const binary = isBinaryContent(fileData);
  const truncated = fileData.truncated === true;

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
        <div className="bg-white border border-[#d0d7de] rounded-lg overflow-hidden">
          <div className="p-3 border-b border-[#d0d7de] flex items-center justify-between gap-3 flex-wrap">
            <div className="flex items-center gap-3 flex-wrap min-w-0">
              <RepoRefSelector
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
                        <span className="font-mono text-[#24292f] font-semibold">{part}</span>
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

            {downloadUrl && (
              <div className="flex gap-2 shrink-0">
                <a
                  href={downloadUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-[#f6f8fa] text-[#24292f] no-underline hover:bg-[#eaeef2]"
                >
                  Raw
                </a>
                <a
                  href={downloadUrl}
                  download={fileData.name}
                  className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-[#f6f8fa] text-[#24292f] no-underline hover:bg-[#eaeef2]"
                >
                  Download
                </a>
              </div>
            )}
          </div>

          {truncated ? (
            <div className="p-8 text-center text-sm text-[#57606a] bg-[#f6f8fa] border-t border-[#d0d7de]">
              This file is too large to display.{" "}
              {downloadUrl && (
                <a
                  href={downloadUrl}
                  download={fileData.name}
                  className="text-[#0969da] hover:underline font-medium"
                >
                  Download file
                </a>
              )}
            </div>
          ) : binary ? (
            <div className="p-8 text-center text-sm text-[#57606a] bg-[#f6f8fa] border-t border-[#d0d7de]">
              Binary file not shown.
              {downloadUrl && (
                <>
                  {" "}
                  <a
                    href={downloadUrl}
                    download={fileData.name}
                    className="text-[#0969da] hover:underline font-medium"
                  >
                    Download file
                  </a>
                </>
              )}
            </div>
          ) : (
            <BlobViewer
              content={
                fileData.content && fileData.encoding === "base64"
                  ? decodeBase64Content(fileData.content)
                  : ""
              }
              filename={fileData.name}
              binary={false}
              truncated={false}
              rawUrl={downloadUrl}
            />
          )}
        </div>
      </div>
    </div>
  );
}
