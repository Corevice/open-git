"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRoot,
  TableRow,
} from "@/components/ui/table";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/lib/auth";

const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  process.env.NEXT_PUBLIC_API_URL ??
  "";

type OrgAuditLogEntry = {
  id: string;
  actor_login: string;
  action: string;
  ip_address: string | null;
  created_at: string;
};

type ExportResponse = {
  job_id: string;
  download_url?: string;
};

const ACTION_OPTIONS = [
  { value: "", label: "All actions" },
  { value: "repo.delete", label: "repo.delete" },
  { value: "member.add", label: "member.add" },
  { value: "member.remove", label: "member.remove" },
  { value: "token.issue", label: "token.issue" },
  { value: "token.revoke", label: "token.revoke" },
  { value: "advisory.state_change", label: "advisory.state_change" },
  { value: "settings.update", label: "settings.update" },
] as const;

function requireAdminRole(token: string | null, router: ReturnType<typeof useRouter>): boolean {
  if (!token) {
    router.push("/login");
    return false;
  }
  return true;
}

export default function SecurityAuditLogPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { token } = useAuth();
  const toast = useToast();

  const [org, setOrg] = useState("");
  const [entries, setEntries] = useState<OrgAuditLogEntry[]>([]);
  const [phrase, setPhrase] = useState("");
  const [action, setAction] = useState("");
  const [after, setAfter] = useState("");
  const [before, setBefore] = useState("");
  const [loading, setLoading] = useState(true);
  const [exporting, setExporting] = useState(false);
  const [accessDenied, setAccessDenied] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const orgParam = searchParams.get("org");

  const loadAuditLog = useCallback(async () => {
    if (!requireAdminRole(token, router)) {
      return;
    }

    setLoading(true);
    setError(null);
    setAccessDenied(false);

    try {
      let resolvedOrg = orgParam ?? "";
      if (!resolvedOrg) {
        const orgsResponse = await fetch(`${API_BASE}/api/v3/user/orgs`, {
          headers: {
            Accept: "application/json",
            Authorization: `Bearer ${token}`,
          },
          cache: "no-store",
        });

        if (orgsResponse.status === 401) {
          router.push("/login");
          return;
        }

        if (!orgsResponse.ok) {
          throw new Error(`Failed to load organizations (${orgsResponse.status})`);
        }

        const orgs = (await orgsResponse.json()) as { login: string }[];
        if (orgs.length === 0) {
          throw new Error("No organization found");
        }
        resolvedOrg = orgs[0].login;
      }

      setOrg(resolvedOrg);

      const params = new URLSearchParams({ per_page: "50" });
      if (phrase.trim()) {
        params.set("phrase", phrase.trim());
      }
      if (action) {
        params.set("include", action);
      }
      if (after) {
        params.set("after", new Date(after).toISOString());
      }
      if (before) {
        params.set("before", new Date(before).toISOString());
      }

      const response = await fetch(
        `${API_BASE}/api/v3/orgs/${encodeURIComponent(resolvedOrg)}/audit-log?${params.toString()}`,
        {
          headers: {
            Accept: "application/json",
            Authorization: `Bearer ${token}`,
          },
          cache: "no-store",
        },
      );

      if (response.status === 401) {
        router.push("/login");
        return;
      }

      if (response.status === 403) {
        setAccessDenied(true);
        setEntries([]);
        return;
      }

      if (!response.ok) {
        throw new Error(`Failed to load audit log (${response.status})`);
      }

      const data = (await response.json()) as OrgAuditLogEntry[];
      setEntries(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load audit log.");
    } finally {
      setLoading(false);
    }
  }, [token, router, orgParam, phrase, action, after, before]);

  useEffect(() => {
    loadAuditLog();
  }, [loadAuditLog]);

  const handleExport = async () => {
    if (!requireAdminRole(token, router) || !org) {
      return;
    }

    setExporting(true);
    try {
      const params = new URLSearchParams({ format: "csv" });
      if (phrase.trim()) {
        params.set("phrase", phrase.trim());
      }
      if (action) {
        params.set("include", action);
      }
      if (after) {
        params.set("after", new Date(after).toISOString());
      }
      if (before) {
        params.set("before", new Date(before).toISOString());
      }

      const response = await fetch(
        `${API_BASE}/api/v3/orgs/${encodeURIComponent(org)}/audit-log/export?${params.toString()}`,
        {
          headers: {
            Accept: "application/json",
            Authorization: `Bearer ${token}`,
          },
        },
      );

      if (response.status === 401) {
        router.push("/login");
        return;
      }

      if (response.status === 403) {
        toast.error("Admin access is required to export audit logs.");
        return;
      }

      if (!response.ok) {
        throw new Error(`Failed to export audit log (${response.status})`);
      }

      const data = (await response.json()) as ExportResponse;
      toast.success(`Export job queued: ${data.job_id}`);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to export audit log.",
      );
    } finally {
      setExporting(false);
    }
  };

  if (accessDenied) {
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
        </header>
        <div className="mx-auto max-w-[1200px] px-6 py-6">
          <h1 className="mb-4 text-2xl font-semibold">Security Audit Log</h1>
          <div className="rounded-md border border-[#d0d7de] bg-white p-6">
            <p className="text-sm font-semibold text-[#cf222e]">Access Denied</p>
            <p className="mt-2 text-sm text-[#656d76]">
              You do not have permission to view the audit log. Admin access is
              required.
            </p>
          </div>
        </div>
      </div>
    );
  }

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
          href="/admin/security"
          className="rounded-md px-3 py-1.5 text-sm hover:bg-[#f6f8fa]"
        >
          ← Security dashboard
        </Link>
      </header>

      <div className="mx-auto max-w-[1200px] px-6 py-6">
        <div className="mb-4 text-sm text-[#656d76]">
          <Link href="/dashboard" className="text-[#0969da]">
            Dashboard
          </Link>{" "}
          /{" "}
          <Link href="/admin/security" className="text-[#0969da]">
            Admin / Security
          </Link>{" "}
          / Audit log
        </div>
        <h1 className="mb-6 text-2xl font-semibold">Security Audit Log</h1>

        <form
          className="mb-4 grid gap-4 rounded-md border border-[#d0d7de] bg-white p-4 md:grid-cols-2 lg:grid-cols-5"
          onSubmit={(event) => {
            event.preventDefault();
            loadAuditLog();
          }}
        >
          <div className="space-y-2 lg:col-span-2">
            <Label htmlFor="phrase">Search phrase</Label>
            <Input
              id="phrase"
              value={phrase}
              onChange={(event) => setPhrase(event.target.value)}
              placeholder="Search actions, actors, targets…"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="action-filter">Action</Label>
            <select
              id="action-filter"
              value={action}
              onChange={(event) => setAction(event.target.value)}
              className="flex h-10 w-full rounded-md border border-[#d0d7de] bg-white px-3 py-2 text-sm"
            >
              {ACTION_OPTIONS.map((option) => (
                <option key={option.value || "all"} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="after">After</Label>
            <Input
              id="after"
              type="datetime-local"
              value={after}
              onChange={(event) => setAfter(event.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="before">Before</Label>
            <Input
              id="before"
              type="datetime-local"
              value={before}
              onChange={(event) => setBefore(event.target.value)}
            />
          </div>
          <div className="flex items-end gap-2 md:col-span-2 lg:col-span-5">
            <Button type="submit" disabled={loading}>
              Search
            </Button>
            <Button
              type="button"
              variant="outline"
              disabled={exporting || !org}
              onClick={() => {
                void handleExport();
              }}
            >
              Export CSV
            </Button>
          </div>
        </form>

        <div className="overflow-hidden rounded-md border border-[#d0d7de] bg-white">
          {loading ? (
            <p className="p-4 text-sm text-[#656d76]">Loading…</p>
          ) : error ? (
            <p className="p-4 text-sm text-[#cf222e]">{error}</p>
          ) : entries.length === 0 ? (
            <p className="p-4 text-sm text-[#656d76]">No audit log entries found.</p>
          ) : (
            <TableRoot>
              <TableHeader>
                <TableRow>
                  <TableHead>Actor</TableHead>
                  <TableHead>Action</TableHead>
                  <TableHead>IP address</TableHead>
                  <TableHead>Created at</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {entries.map((entry) => (
                  <TableRow key={entry.id}>
                    <TableCell className="font-mono text-xs">
                      {entry.actor_login}
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {entry.action}
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {entry.ip_address ?? "—"}
                    </TableCell>
                    <TableCell className="text-xs text-[#656d76]">
                      {entry.created_at}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </TableRoot>
          )}
        </div>
      </div>
    </div>
  );
}
