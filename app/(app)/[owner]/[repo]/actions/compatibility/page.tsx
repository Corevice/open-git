"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { CompatBadge } from "@/components/ui/compat-badge";
import type { ActionCompatibilityResult } from "@/lib/api-types";

export default function CompatibilityPage() {
  const params = useParams();
  const owner = params.owner as string;
  const repo = params.repo as string;

  const [results, setResults] = useState<ActionCompatibilityResult[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadResults = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/repos/${owner}/${repo}/actions/compatibility`);
      if (!res.ok) throw new Error("Failed to load compatibility results");
      const data = (await res.json()) as ActionCompatibilityResult[];
      setResults(data ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load compatibility results");
    } finally {
      setLoading(false);
    }
  }, [owner, repo]);

  useEffect(() => {
    loadResults();
  }, [loadResults]);

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <div className="max-w-[1280px] mx-auto px-6 py-6">
        <h1 className="text-2xl font-semibold mb-6">
          <span className="text-[#0969da]">{owner}</span> /{" "}
          <span className="text-[#0969da]">{repo}</span>
          <span className="ml-2 text-lg font-normal text-[#656d76]">Compatibility</span>
        </h1>

        {error && (
          <div className="mb-4 rounded-md border border-[#cf222e] bg-[#ffebe9] px-4 py-3 text-sm text-[#cf222e]">
            {error}
          </div>
        )}

        <div className="bg-white border border-[#d0d7de] rounded-md overflow-hidden">
          <div className="px-4 py-3 bg-[#f6f8fa] border-b border-[#d0d7de] text-sm text-[#656d76]">
            {loading ? (
              "Loading…"
            ) : (
              <span>
                <strong>{results.length}</strong> compatibility results
              </span>
            )}
          </div>

          {loading ? (
            <div className="px-4 py-8 text-center text-[#656d76]">
              Loading compatibility results…
            </div>
          ) : results.length === 0 ? (
            <div className="px-4 py-8 text-center text-[#656d76]">
              No compatibility results yet.
            </div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-[#d0d7de] bg-[#f6f8fa] text-left text-xs text-[#656d76]">
                  <th className="px-4 py-2 font-medium">Action</th>
                  <th className="px-4 py-2 font-medium">Version</th>
                  <th className="px-4 py-2 font-medium">Status</th>
                  <th className="px-4 py-2 font-medium">Note</th>
                  <th className="px-4 py-2 font-medium">Last Verified</th>
                </tr>
              </thead>
              <tbody>
                {results.map((r) => (
                  <tr
                    key={`${r.action}-${r.version}`}
                    className="border-b border-[#d8dee4] last:border-b-0 hover:bg-[#fafbfc]"
                  >
                    <td className="px-4 py-3 font-mono">{r.action}</td>
                    <td className="px-4 py-3 font-mono text-xs">{r.version}</td>
                    <td className="px-4 py-3">
                      <CompatBadge status={r.status} />
                    </td>
                    <td className="px-4 py-3 text-[#656d76]">{r.note ?? "—"}</td>
                    <td className="px-4 py-3 text-[#656d76]">
                      {r.last_verified_at
                        ? new Date(r.last_verified_at).toLocaleDateString()
                        : "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        <Link
          href={`/${owner}/${repo}/actions`}
          className="text-sm text-[#0969da] hover:underline mt-4 inline-block"
        >
          ← Back to Actions
        </Link>
      </div>
    </div>
  );
}
