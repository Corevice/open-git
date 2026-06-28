"use client";

import type { Artifact } from "@/lib/api/actions";

type ArtifactListProps = {
  owner: string;
  repo: string;
  artifacts: Artifact[];
  loading: boolean;
};

export function formatSize(sizeInBytes: number): string {
  if (sizeInBytes < 1024) {
    return `${sizeInBytes} bytes`;
  }
  if (sizeInBytes < 1048576) {
    return `${(sizeInBytes / 1024).toFixed(1)} KB`;
  }
  return `${(sizeInBytes / 1048576).toFixed(1)} MB`;
}

function formatExpiresDate(expiresAt: string): string {
  return new Date(expiresAt).toLocaleDateString();
}

export function ArtifactList({ owner, repo, artifacts, loading }: ArtifactListProps) {
  if (loading) {
    return (
      <div className="space-y-3" aria-busy="true" aria-label="Loading artifacts">
        {[0, 1, 2].map((key) => (
          <div
            key={key}
            className="h-12 animate-pulse rounded-md border border-[#d0d7de] bg-[#f6f8fa]"
          />
        ))}
      </div>
    );
  }

  if (artifacts.length === 0) {
    return (
      <p className="text-sm text-[#656d76]">No artifacts produced by this run</p>
    );
  }

  return (
    <ul className="divide-y divide-[#d0d7de] border border-[#d0d7de] rounded-md overflow-hidden">
      {artifacts.map((artifact) => {
        const downloadHref = `/api/repos/${owner}/${repo}/actions/artifacts/${artifact.id}/zip`;

        return (
          <li
            key={artifact.id}
            className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 bg-white text-sm"
          >
            <div className="min-w-0">
              <div className="font-medium text-[#1f2328]">{artifact.name}</div>
              <div className="text-xs text-[#656d76]">
                {formatSize(artifact.size_in_bytes)} · Expires {formatExpiresDate(artifact.expires_at)}
              </div>
            </div>
            <a
              href={downloadHref}
              target="_blank"
              rel="noopener noreferrer"
              aria-disabled={artifact.expired ? "true" : undefined}
              title={artifact.expired ? "Artifact has expired" : undefined}
              className={`text-sm font-medium ${
                artifact.expired
                  ? "text-[#656d76] cursor-not-allowed pointer-events-none"
                  : "text-[#0969da] hover:underline"
              }`}
              onClick={
                artifact.expired
                  ? (event) => {
                      event.preventDefault();
                    }
                  : undefined
              }
            >
              Download
            </a>
          </li>
        );
      })}
    </ul>
  );
}
