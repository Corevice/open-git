import Link from "next/link";
import { notFound } from "next/navigation";
import BlobViewer from "@/components/repo/BlobViewer";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";

interface ContentResponse {
  name: string;
  path: string;
  sha: string;
  size: number;
  type: "file";
  content?: string;
  encoding?: string;
  download_url?: string;
  truncated?: boolean;
  binary?: boolean;
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

function decodeContent(data: ContentResponse): string {
  if (!data.content || data.encoding !== "base64") return "";
  try {
    return Buffer.from(data.content.replace(/\n/g, ""), "base64").toString("utf-8");
  } catch {
    return "";
  }
}

export default async function BlobPage({
  params,
}: {
  params: Promise<{ owner: string; repo: string; branch: string; path: string[] }>;
}) {
  const { owner, repo, branch, path: pathSegments } = await params;
  const filePath = pathSegments.join("/");

  const metadata = await apiGet<{ default_branch: string }>(`/repos/${owner}/${repo}`);
  if (!metadata) notFound();

  const contentData = await apiGet<ContentResponse>(
    `/repos/${owner}/${repo}/contents/${filePath}?ref=${encodeURIComponent(branch)}`,
  );
  if (!contentData || contentData.type !== "file") notFound();

  const content = decodeContent(contentData);
  const rawUrl = `${API_BASE}/repos/${owner}/${repo}/contents/${filePath}?ref=${encodeURIComponent(branch)}`;
  const downloadUrl = contentData.download_url ?? rawUrl;
  const pathParts = filePath.split("/");

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

            <div className="flex gap-2 shrink-0">
              <a
                href={rawUrl}
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
          </div>

          <BlobViewer
            content={content}
            filename={contentData.name}
            binary={contentData.binary}
            truncated={contentData.truncated}
            rawUrl={rawUrl}
          />
        </div>
      </div>
    </div>
  );
}
