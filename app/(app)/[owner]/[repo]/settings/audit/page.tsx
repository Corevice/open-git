"use client";

import { use, useCallback, useEffect, useState } from "react";
import Link from "next/link";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";

interface AuditEntry {
  id: string | number;
  actor_login: string;
  action: string;
  target_type: string;
  target_id: string | number;
  created_at: string;
  metadata?: Record<string, unknown>;
}

interface PageLinks {
  next?: string;
  prev?: string;
  first?: string;
  last?: string;
}

const ACTION_FILTERS = [
  { value: "", label: "All actions" },
  { value: "repo.create", label: "repo.create" },
  { value: "repo.delete", label: "repo.delete" },
  { value: "repo.visibility_change", label: "repo.visibility_change" },
  { value: "pr.merge", label: "pr.merge" },
  { value: "branch_protection.update", label: "branch_protection.update" },
  { value: "webhook.create", label: "webhook.create" },
  { value: "token.issue", label: "token.issue" },
  { value: "token.revoke", label: "token.revoke" },
  { value: "settings.update", label: "settings.update" },
] as const;

function parseLinkHeader(header: string | null): PageLinks {
  if (!header) return {};
  const links: PageLinks = {};
  header.split(",").forEach((part) => {
    const match = part.match(/<([^>]+)>;\s*rel="([^"]+)"/);
    if (!match) return;
    const [, url, rel] = match;
    if (rel === "next" || rel === "prev" || rel === "first" || rel === "last") {
      links[rel] = url;
    }
  });
  return links;
}

function urlPathAndQuery(absoluteOrRelative: string): string {
  try {
    const u = new URL(absoluteOrRelative, "http://placeholder.invalid");
    return `${u.pathname}${u.search}`;
  } catch {
    return absoluteOrRelative;
  }
}

export default function AuditLogPage({
  params,
}: {
  params: Promise<{ owner: string; repo: string }>;
}) {
  const { owner, repo } = use(params);
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [links, setLinks] = useState<PageLinks>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [action, setAction] = useState("");
  const [currentPath, setCurrentPath] = useState<string | null>(null);

  const load = useCallback(
    async (path?: string) => {
      setLoading(true);
      setError(null);
      try {
        const url =
          path ??
          `/repos/${owner}/${repo}/audit-log${action ? `?action=${encodeURIComponent(action)}` : ""}`;
        const res = await fetch(`${API_BASE}${url}`, {
          headers: { Accept: "application/vnd.github+json" },
          cache: "no-store",
        });
        if (!res.ok) {
          throw new Error(`Failed to load audit log (${res.status})`);
        }
        const data = (await res.json()) as AuditEntry[];
        setEntries(data);
        setLinks(parseLinkHeader(res.headers.get("Link")));
        setCurrentPath(url);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load.");
      } finally {
        setLoading(false);
      }
    },
    [owner, repo, action],
  );

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <header className="sticky top-0 z-50 flex h-16 items-center justify-between border-b border-[#d1d9e0] bg-white/85 px-6 backdrop-blur">
        <Link
          href="/dashboard"
          className="flex items-center gap-2 text-lg font-extrabold"
        >
          <span className="text-xl">🐙</span>
          <span>OpenHub</span>
        </Link>
        <Link
          href={`/${owner}/${repo}/settings`}
          className="rounded-md px-3 py-1.5 text-sm hover:bg-[#f6f8fa]"
        >
          ← Settings
        </Link>
      </header>

      <div className="mx-auto max-w-[1200px] px-6 py-6">
        <div className="mb-4 text-sm text-[#656d76]">
          <Link href={`/${owner}/${repo}`} className="text-[#0969da]">
            {owner}/{repo}
          </Link>{" "}
          /{" "}
          <Link
            href={`/${owner}/${repo}/settings`}
            className="text-[#0969da]"
          >
            Settings
          </Link>{" "}
          / Audit log
        </div>
        <h1 className="mb-4 text-2xl font-semibold">Audit log</h1>

        <div className="mb-4 flex items-center gap-3 rounded-md border border-[#d0d7de] bg-white p-3">
          <label
            htmlFor="action-filter"
            className="text-sm font-semibold text-[#1f2328]"
          >
            Action:
          </label>
          <select
            id="action-filter"
            value={action}
            onChange={(e) => setAction(e.target.value)}
            className="rounded-md border border-[#d0d7de] bg-white px-3 py-1.5 text-sm"
          >
            {ACTION_FILTERS.map((f) => (
              <option key={f.value} value={f.value}>
                {f.label}
              </option>
            ))}
          </select>
          {currentPath && (
            <span className="ml-auto text-xs text-[#656d76] font-mono truncate max-w-[40%]">
              {currentPath}
            </span>
          )}
        </div>

        <div className="overflow-hidden rounded-md border border-[#d0d7de] bg-white">
          {loading ? (
            <p className="p-4 text-sm text-[#656d76]">Loading…</p>
          ) : error ? (
            <p className="p-4 text-sm text-[#cf222e]">{error}</p>
          ) : entries.length === 0 ? (
            <p className="p-4 text-sm text-[#656d76]">No audit log entries.</p>
          ) : (
            <table className="w-full table-auto text-sm">
              <thead className="border-b border-[#d0d7de] bg-[#f6f8fa] text-left text-xs uppercase text-[#656d76]">
                <tr>
                  <th className="px-4 py-2">Actor</th>
                  <th className="px-4 py-2">Action</th>
                  <th className="px-4 py-2">Target</th>
                  <th className="px-4 py-2">Created at</th>
                </tr>
              </thead>
              <tbody>
                {entries.map((entry) => (
                  <tr
                    key={entry.id}
                    className="border-b border-[#eaeef2] last:border-b-0"
                  >
                    <td className="px-4 py-2 font-mono text-xs">
                      {entry.actor_login}
                    </td>
                    <td className="px-4 py-2 font-mono text-xs">
                      {entry.action}
                    </td>
                    <td className="px-4 py-2 font-mono text-xs">
                      {entry.target_type}#{String(entry.target_id)}
                    </td>
                    <td className="px-4 py-2 text-xs text-[#656d76]">
                      {entry.created_at}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        <div className="mt-4 flex items-center justify-end gap-2">
          <button
            type="button"
            onClick={() =>
              links.prev && load(urlPathAndQuery(links.prev))
            }
            disabled={!links.prev || loading}
            className="rounded-md border border-[#d0d7de] bg-white px-3 py-1.5 text-sm hover:bg-[#f6f8fa] disabled:opacity-50"
          >
            ← Previous
          </button>
          <button
            type="button"
            onClick={() =>
              links.next && load(urlPathAndQuery(links.next))
            }
            disabled={!links.next || loading}
            className="rounded-md border border-[#d0d7de] bg-white px-3 py-1.5 text-sm hover:bg-[#f6f8fa] disabled:opacity-50"
          >
            Next →
          </button>
        </div>
      </div>
    </div>
  );
}
