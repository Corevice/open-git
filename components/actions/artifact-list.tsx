"use client";

import { Archive, Download, Loader2, Trash2 } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

export type Artifact = {
  id: string;
  name: string;
  size_in_bytes: number;
  created_at: string;
  expires_at: string;
  expired: boolean;
};

export type ArtifactsResponse = {
  total_count: number;
  artifacts: Artifact[];
};

type ArtifactListProps = {
  owner: string;
  repo: string;
  runId: string;
};

export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(k)), sizes.length - 1);
  const value = bytes / Math.pow(k, i);
  return `${value >= 10 || i === 0 ? Math.round(value) : value.toFixed(1)} ${sizes[i]}`;
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) return dateStr;
  return date.toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function isArtifactsResponse(data: unknown): data is ArtifactsResponse {
  if (typeof data !== "object" || data === null) return false;
  const record = data as Record<string, unknown>;
  return typeof record.total_count === "number" && Array.isArray(record.artifacts);
}

function isArtifact(value: unknown): value is Artifact {
  if (typeof value !== "object" || value === null) return false;
  const record = value as Record<string, unknown>;
  return (
    typeof record.id === "string" &&
    typeof record.name === "string" &&
    typeof record.size_in_bytes === "number" &&
    typeof record.created_at === "string" &&
    typeof record.expires_at === "string" &&
    typeof record.expired === "boolean"
  );
}

export default function ArtifactList({ owner, repo, runId }: ArtifactListProps) {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [downloadingId, setDownloadingId] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const fetchArtifacts = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(
        `/api/v3/repos/${owner}/${repo}/actions/runs/${runId}/artifacts`,
      );
      if (!res.ok) {
        throw new Error(`Failed to load artifacts (${res.status})`);
      }
      const data: unknown = await res.json();
      if (!isArtifactsResponse(data)) {
        throw new Error("Invalid artifacts response");
      }
      const validArtifacts = data.artifacts.filter(isArtifact);
      setTotalCount(data.total_count);
      setArtifacts(validArtifacts);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load artifacts");
      setTotalCount(0);
      setArtifacts([]);
    } finally {
      setLoading(false);
    }
  }, [owner, repo, runId]);

  useEffect(() => {
    void fetchArtifacts();
  }, [fetchArtifacts]);

  async function handleDownload(artifact: Artifact) {
    if (artifact.expired) return;

    setDownloadingId(artifact.id);
    try {
      const res = await fetch(
        `/api/v3/repos/${owner}/${repo}/actions/artifacts/${artifact.id}/zip`,
        { redirect: "manual" },
      );

      if (res.status === 302 || res.status === 301) {
        const location = res.headers.get("Location");
        if (location) {
          window.location.href = location;
          return;
        }
        throw new Error("Download redirect missing location");
      }

      if (!res.ok) {
        throw new Error(`Download failed (${res.status})`);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Download failed");
    } finally {
      setDownloadingId(null);
    }
  }

  async function handleDelete(artifact: Artifact) {
    if (!window.confirm(`Delete artifact "${artifact.name}"?`)) return;

    setDeletingId(artifact.id);
    setError(null);
    try {
      const res = await fetch(
        `/api/v3/repos/${owner}/${repo}/actions/artifacts/${artifact.id}`,
        { method: "DELETE" },
      );
      if (!res.ok) {
        throw new Error(`Failed to delete artifact (${res.status})`);
      }
      await fetchArtifacts();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to delete artifact");
    } finally {
      setDeletingId(null);
    }
  }

  return (
    <section className="mt-6 bg-white border border-[#d0d7de] rounded-md overflow-hidden">
      <div className="flex items-center gap-2 px-4 py-3 bg-[#f6f8fa] border-b border-[#d0d7de] text-sm font-semibold">
        <Archive className="h-4 w-4" aria-hidden />
        Artifacts
      </div>

      {loading && (
        <div className="flex items-center gap-2 px-4 py-8 text-sm text-[#656d76]">
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
          Loading artifacts…
        </div>
      )}

      {!loading && error && (
        <div
          role="alert"
          className="mx-4 my-4 rounded-md border border-[#cf222e] bg-[#ffebe9] px-4 py-3 text-sm text-[#cf222e]"
        >
          {error}
        </div>
      )}

      {!loading && !error && totalCount === 0 && (
        <div className="px-4 py-8 text-sm text-[#656d76]">No artifacts for this run</div>
      )}

      {!loading && !error && totalCount > 0 && (
        <ul className="divide-y divide-[#d8dee4]">
          {artifacts.map((artifact) => (
            <li
              key={artifact.id}
              className="flex flex-wrap items-center gap-4 px-4 py-3 text-sm"
            >
              <div className="min-w-0 flex-1">
                <div className="font-medium text-[#1f2328] truncate">{artifact.name}</div>
                <div className="mt-1 flex flex-wrap gap-x-4 gap-y-1 text-xs text-[#656d76]">
                  <span>{formatBytes(artifact.size_in_bytes)}</span>
                  <span>Created {formatDate(artifact.created_at)}</span>
                  <span>Expires {formatDate(artifact.expires_at)}</span>
                </div>
              </div>

              <div className="flex items-center gap-2">
                {artifact.expired && (
                  <span className="rounded-full bg-[#eaeef2] px-2 py-0.5 text-xs font-medium text-[#656d76]">
                    Expired
                  </span>
                )}

                <button
                  type="button"
                  onClick={() => void handleDownload(artifact)}
                  disabled={artifact.expired || downloadingId === artifact.id}
                  className="inline-flex items-center gap-1.5 rounded-md border border-[#d0d7de] bg-[#f6f8fa] px-3 py-1.5 text-xs font-medium text-[#1f2328] hover:bg-[#eaeef2] disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {downloadingId === artifact.id ? (
                    <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden />
                  ) : (
                    <Download className="h-3.5 w-3.5" aria-hidden />
                  )}
                  Download
                </button>

                <button
                  type="button"
                  onClick={() => void handleDelete(artifact)}
                  disabled={deletingId === artifact.id}
                  className="inline-flex items-center gap-1.5 rounded-md border border-[#d0d7de] bg-white px-3 py-1.5 text-xs font-medium text-[#cf222e] hover:bg-[#ffebe9] disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {deletingId === artifact.id ? (
                    <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden />
                  ) : (
                    <Trash2 className="h-3.5 w-3.5" aria-hidden />
                  )}
                  Delete
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
