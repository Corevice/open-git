"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useState } from "react";

import {
  SecurityAccessDenied,
  SecurityPageLayout,
} from "@/components/admin/SecurityPageLayout";
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
import {
  checkClientOrgAdminAccess,
  datetimeLocalToIso,
  genericActionError,
  getSecurityApiBase,
  maskIpAddress,
  resolveOrgLogin,
  sanitizeAuditSearchPhrase,
} from "@/lib/admin/security";
import { useAuth } from "@/lib/auth";

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

function SecurityAuditLogContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { token } = useAuth();
  const toast = useToast();
  const apiBase = getSecurityApiBase();

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
    setLoading(true);
    setError(null);
    setAccessDenied(false);

    const access = await checkClientOrgAdminAccess({
      token,
      orgParam,
      onUnauthenticated: () => {
        router.push("/login");
      },
    });

    if (access.status === "unauthenticated") {
      setLoading(false);
      return;
    }

    if (access.status === "access_denied") {
      setAccessDenied(true);
      setEntries([]);
      setLoading(false);
      return;
    }

    if (access.status === "error") {
      setError(access.message);
      setLoading(false);
      return;
    }

    const resolvedOrg = access.org;
    setOrg(resolvedOrg);

    try {
      const params = new URLSearchParams({ per_page: "50" });
      const sanitizedPhrase = sanitizeAuditSearchPhrase(phrase);
      if (sanitizedPhrase) {
        params.set("phrase", sanitizedPhrase);
      }
      if (action) {
        params.set("action", action);
      }
      const afterIso = datetimeLocalToIso(after);
      if (afterIso) {
        params.set("after", afterIso);
      }
      const beforeIso = datetimeLocalToIso(before);
      if (beforeIso) {
        params.set("before", beforeIso);
      }

      const response = await fetch(
        `${apiBase}/api/v3/orgs/${encodeURIComponent(resolvedOrg)}/audit-log?${params.toString()}`,
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
        throw new Error(genericActionError("load audit log"));
      }

      const data = (await response.json()) as OrgAuditLogEntry[];
      setEntries(data);
    } catch {
      setError(genericActionError("load audit log"));
    } finally {
      setLoading(false);
    }
  }, [token, router, orgParam, phrase, action, after, before, apiBase]);

  useEffect(() => {
    void loadAuditLog();
  }, [loadAuditLog]);

  const handleExport = async () => {
    if (!token) {
      router.push("/login");
      return;
    }

    setExporting(true);
    try {
      let resolvedOrg = org;
      if (!resolvedOrg) {
        const orgResult = await resolveOrgLogin(token, orgParam);
        if (!orgResult.ok) {
          toast.error(orgResult.message);
          return;
        }
        resolvedOrg = orgResult.org;
        setOrg(resolvedOrg);
      }

      const access = await checkClientOrgAdminAccess({
        token,
        orgParam: resolvedOrg,
        onUnauthenticated: () => {
          router.push("/login");
        },
      });

      if (access.status !== "ok") {
        toast.error("Admin access is required to export audit logs.");
        return;
      }

      const params = new URLSearchParams({ format: "csv" });
      const sanitizedPhrase = sanitizeAuditSearchPhrase(phrase);
      if (sanitizedPhrase) {
        params.set("phrase", sanitizedPhrase);
      }
      if (action) {
        params.set("action", action);
      }
      const afterIso = datetimeLocalToIso(after);
      if (afterIso) {
        params.set("after", afterIso);
      }
      const beforeIso = datetimeLocalToIso(before);
      if (beforeIso) {
        params.set("before", beforeIso);
      }

      const response = await fetch(
        `${apiBase}/api/v3/orgs/${encodeURIComponent(resolvedOrg)}/audit-log/export?${params.toString()}`,
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
        throw new Error(genericActionError("export audit log"));
      }

      const data = (await response.json()) as ExportResponse;
      toast.success(`Export job queued: ${data.job_id}`);
    } catch {
      toast.error(genericActionError("export audit log"));
    } finally {
      setExporting(false);
    }
  };

  if (accessDenied) {
    return (
      <SecurityAccessDenied
        title="Security Audit Log"
        breadcrumbSuffix="Audit log"
      />
    );
  }

  return (
    <SecurityPageLayout
      title="Security Audit Log"
      breadcrumbSuffix="Audit log"
      backHref="/admin/security"
      backLabel="← Security dashboard"
    >
      <form
        className="mb-4 grid gap-4 rounded-md border border-[#d0d7de] bg-white p-4 md:grid-cols-2 lg:grid-cols-5"
        onSubmit={(event) => {
          event.preventDefault();
          void loadAuditLog();
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
            disabled={exporting || loading}
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
                    {maskIpAddress(entry.ip_address)}
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
    </SecurityPageLayout>
  );
}

export default function SecurityAuditLogPage() {
  return (
    <Suspense fallback={null}>
      <SecurityAuditLogContent />
    </Suspense>
  );
}
