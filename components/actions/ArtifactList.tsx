"use client";

import type { Artifact } from "@/lib/api/actions";

type ArtifactListProps = {
  owner: string;
  repo: string;
  artifacts: Artifact[];
  loading: boolean;
};

function formatBytes(bytes: number): string {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const exponent = Math.min(
    units.length - 1,
    Math.floor(Math.log(bytes) / Math.log(1024)),
  );
  const value = bytes / Math.pow(1024, exponent);
  return `${value.toFixed(exponent === 0 ? 0 : 1)} ${units[exponent]}`;
}

export function ArtifactList({
  owner,
  repo,
  artifacts,
  loading,
}: ArtifactListProps) {
  return (
    <section className="rounded-md border border-[#d0d7de] bg-white">
      <h2 className="border-b border-[#d0d7de] px-4 py-3 text-sm font-semibold text-[#1f2328]">
        Artifacts
      </h2>

      {loading ? (
        <p className="px-4 py-6 text-sm text-[#656d76]">Loading artifacts…</p>
      ) : artifacts.length === 0 ? (
        <p className="px-4 py-6 text-sm text-[#656d76]">
          No artifacts were produced by this run.
        </p>
      ) : (
        <ul className="divide-y divide-[#d8dee4]">
          {artifacts.map((artifact) => {
            const downloadUrl =
              artifact.archive_download_url ??
              `/api/v3/repos/${owner}/${repo}/actions/artifacts/${artifact.id}/zip`;

            return (
              <li
                key={artifact.id}
                className="flex items-center justify-between px-4 py-3 text-sm"
              >
                <div>
                  <p className="font-medium text-[#1f2328]">{artifact.name}</p>
                  <p className="text-xs text-[#656d76]">
                    {formatBytes(artifact.size_in_bytes)}
                    {artifact.expired ? " · expired" : ""}
                  </p>
                </div>
                {artifact.expired ? (
                  <span className="text-xs text-[#656d76]">Expired</span>
                ) : (
                  <a
                    href={downloadUrl}
                    className="text-sm text-[#0969da] hover:underline"
                  >
                    Download
                  </a>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
