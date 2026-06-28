import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import BlobViewer from "@/components/repo/BlobViewer";
import { Skeleton } from "@/components/ui/skeleton";
import {
  apiClient,
  decodeBase64Content,
  decodePathSegments,
  isApiError,
} from "@/lib/api-client";

interface ContentResponse {
  name: string;
  path: string;
  sha: string;
  size: number;
  type: "file";
  content?: string | null;
  encoding?: string;
  download_url?: string;
  truncated?: boolean;
}

function isBinaryContent(data: ContentResponse): boolean {
  return data.content == null;
}

async function BlobViewerContent({
  owner,
  repo,
  branch,
  decodedPath,
}: {
  owner: string;
  repo: string;
  branch: string;
  decodedPath: string;
}) {
  let contentData: ContentResponse;
  try {
    contentData = await apiClient.getContents<ContentResponse>(
      owner,
      repo,
      decodedPath,
      branch,
    );
  } catch (err) {
    if (isApiError(err) && err.status === 404) notFound();
    throw err;
  }

  if (contentData.type !== "file") notFound();

  const downloadUrl = contentData.download_url ?? "";
  const binary = isBinaryContent(contentData);
  const truncated = contentData.truncated === true;

  return (
    <>
      {downloadUrl && (
        <div className="p-3 border-b border-[#d0d7de] flex justify-end gap-2">
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
            download={contentData.name}
            className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-[#f6f8fa] text-[#24292f] no-underline hover:bg-[#eaeef2]"
          >
            Download
          </a>
        </div>
      )}

      {truncated ? (
        <div className="p-8 text-center text-sm text-[#57606a] bg-[#f6f8fa] border-t border-[#d0d7de]">
          This file is too large to display.{" "}
          {downloadUrl && (
            <a
              href={downloadUrl}
              download={contentData.name}
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
                download={contentData.name}
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
            contentData.content && contentData.encoding === "base64"
              ? decodeBase64Content(contentData.content)
              : ""
          }
          filename={contentData.name}
          binary={false}
          truncated={false}
          rawUrl={downloadUrl}
        />
      )}
    </>
  );
}

export default async function BlobPage({
  params,
}: {
  params: Promise<{ owner: string; repo: string; branch: string; path: string[] }>;
}) {
  const { owner, repo, branch: rawBranch, path: rawPathSegments } = await params;
  const branch = decodeURIComponent(rawBranch);
  const decodedPath = decodePathSegments(rawPathSegments ?? []).join("/");
  const pathParts = decodedPath.split("/");

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
            <nav className="flex items-center gap-1 text-sm flex-wrap min-w-0">
              <Link href={`/${owner}/${repo}`} className="text-[#0969da] hover:underline no-underline">
                {repo}
              </Link>
              <span className="text-[#57606a]">/</span>
              <Link
                href={`/${owner}/${repo}/tree/${encodeURIComponent(branch)}`}
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
                      <span className="font-mono text-[#24292f] font-semibold">{part}</span>
                    ) : (
                      <Link
                        href={`/${owner}/${repo}/tree/${encodeURIComponent(branch)}/${sub
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

          <Suspense
            fallback={
              <div className="space-y-2 p-4">
                {[...Array(8)].map((_, i) => (
                  <Skeleton key={i} className="h-4 w-full" />
                ))}
              </div>
            }
          >
            <BlobViewerContent
              owner={owner}
              repo={repo}
              branch={branch}
              decodedPath={decodedPath}
            />
          </Suspense>
        </div>
      </div>
    </div>
  );
}
