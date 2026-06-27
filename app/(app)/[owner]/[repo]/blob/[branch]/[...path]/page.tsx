import Link from "next/link";
import { notFound } from "next/navigation";
import BlobViewer from "@/components/repo/BlobViewer";
import { apiClient, type ApiError } from "@/lib/api-client";

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

type RepoApiClient = typeof apiClient & {
  getContents(
    owner: string,
    repo: string,
    path: string,
    ref: string,
  ): Promise<ContentResponse>;
};

const client = apiClient as RepoApiClient;

function isBinaryContent(data: ContentResponse): boolean {
  return data.content == null;
}

function decodeContent(data: ContentResponse): string {
  if (!data.content || data.encoding !== "base64") return "";
  try {
    return Buffer.from(data.content.replace(/\n/g, ""), "base64").toString("utf-8");
  } catch {
    return "";
  }
}

function isNotFound(err: unknown): boolean {
  return (err as ApiError).status === 404;
}

export default async function BlobPage({
  params,
}: {
  params: Promise<{ owner: string; repo: string; branch: string; path: string[] }>;
}) {
  const { owner, repo, branch, path: pathSegments } = await params;
  const decodedPath = pathSegments.join("/");

  let contentData: ContentResponse;
  try {
    contentData = await client.getContents(owner, repo, decodedPath, branch);
  } catch (err) {
    if (isNotFound(err)) notFound();
    throw err;
  }

  if (contentData.type !== "file") notFound();

  const rawUrl = contentData.download_url ?? "";
  const downloadUrl = contentData.download_url ?? rawUrl;
  const pathParts = decodedPath.split("/");
  const binary = isBinaryContent(contentData);
  const truncated = contentData.truncated === true;

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
                      <span className="font-mono text-[#24292f] font-semibold">{part}</span>
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
                  download={contentData.name}
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
              content={decodeContent(contentData)}
              filename={contentData.name}
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
