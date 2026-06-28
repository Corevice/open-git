"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState, type ReactNode } from "react";

import { AdvisoryStatusForm } from "@/components/admin/AdvisoryStatusForm";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import type {
  AdvisorySeverity,
  AdvisoryState,
  DismissedReason,
  SecurityAdvisory,
} from "@/lib/api-types";
import { cn } from "@/lib/utils";

const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  process.env.NEXT_PUBLIC_API_URL ??
  "";

type AdvisoryListItem = SecurityAdvisory & {
  repository?: {
    owner: { login: string };
    name: string;
  };
};

const STATE_OPTIONS: { value: string; label: string }[] = [
  { value: "", label: "All states" },
  { value: "open", label: "Open" },
  { value: "acknowledged", label: "Acknowledged" },
  { value: "resolved", label: "Resolved" },
  { value: "dismissed", label: "Dismissed" },
];

const SEVERITY_OPTIONS: { value: string; label: string }[] = [
  { value: "", label: "All severities" },
  { value: "critical", label: "Critical" },
  { value: "high", label: "High" },
  { value: "medium", label: "Medium" },
  { value: "low", label: "Low" },
];

const severityBadgeClass: Record<AdvisorySeverity, string> = {
  critical: "border-transparent bg-[#cf222e] text-white",
  high: "border-transparent bg-[#ffebe9] text-[#cf222e]",
  medium: "border-transparent bg-[#fff8c5] text-[#9a6700]",
  low: "border-transparent bg-[#eaeef2] text-[#656d76]",
};

function Dialog({
  open,
  onOpenChange,
  children,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: ReactNode;
}) {
  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button
        type="button"
        aria-label="Close dialog"
        className="fixed inset-0 bg-black/50"
        onClick={() => onOpenChange(false)}
      />
      <div
        role="dialog"
        aria-modal="true"
        className="relative z-10 w-full max-w-md rounded-md border border-[#d0d7de] bg-white p-6 shadow-lg"
      >
        {children}
      </div>
    </div>
  );
}

function requireAdminRole(token: string | null, router: ReturnType<typeof useRouter>): boolean {
  if (!token) {
    router.push("/login");
    return false;
  }
  return true;
}

function resolveRepoScope(advisory: AdvisoryListItem): {
  owner: string;
  repo: string;
} | null {
  if (advisory.repository?.owner.login && advisory.repository.name) {
    return {
      owner: advisory.repository.owner.login,
      repo: advisory.repository.name,
    };
  }
  return null;
}

export default function SecurityAdvisoriesPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { token } = useAuth();
  const toast = useToast();

  const [org, setOrg] = useState("");
  const [advisories, setAdvisories] = useState<AdvisoryListItem[]>([]);
  const [stateFilter, setStateFilter] = useState("");
  const [severityFilter, setSeverityFilter] = useState("");
  const [loading, setLoading] = useState(true);
  const [accessDenied, setAccessDenied] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedAdvisory, setSelectedAdvisory] =
    useState<AdvisoryListItem | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const orgParam = searchParams.get("org");

  const loadAdvisories = useCallback(async () => {
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

      const params = new URLSearchParams({ per_page: "100" });
      if (stateFilter) {
        params.set("state", stateFilter);
      }
      if (severityFilter) {
        params.set("severity", severityFilter);
      }

      const response = await fetch(
        `${API_BASE}/api/v3/orgs/${encodeURIComponent(resolvedOrg)}/security-advisories?${params.toString()}`,
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
        setAdvisories([]);
        return;
      }

      if (!response.ok) {
        throw new Error(`Failed to load advisories (${response.status})`);
      }

      const data = (await response.json()) as AdvisoryListItem[];
      setAdvisories(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load advisories.");
    } finally {
      setLoading(false);
    }
  }, [token, router, orgParam, stateFilter, severityFilter]);

  useEffect(() => {
    loadAdvisories();
  }, [loadAdvisories]);

  const handleStatusUpdate = async (
    state: AdvisoryState,
    dismissedReason?: DismissedReason,
  ) => {
    if (!selectedAdvisory || !token) {
      return;
    }

    const repoScope = resolveRepoScope(selectedAdvisory);
    if (!repoScope) {
      toast.error("Repository scope is missing for this advisory.");
      return;
    }

    setSubmitting(true);
    try {
      const body: { state: AdvisoryState; dismissed_reason?: DismissedReason } =
        { state };
      if (state === "dismissed" && dismissedReason) {
        body.dismissed_reason = dismissedReason;
      }

      const response = await fetch(
        `${API_BASE}/api/v3/repos/${encodeURIComponent(repoScope.owner)}/${encodeURIComponent(repoScope.repo)}/security-advisories/${encodeURIComponent(selectedAdvisory.ghsa_id)}`,
        {
          method: "PATCH",
          headers: {
            Accept: "application/json",
            "Content-Type": "application/json",
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify(body),
        },
      );

      if (response.status === 401) {
        router.push("/login");
        return;
      }

      if (response.status === 403) {
        toast.error("Admin access is required to update advisory status.");
        return;
      }

      if (!response.ok) {
        let message = `Failed to update advisory (${response.status})`;
        try {
          const errorBody = (await response.json()) as { message?: string };
          message = errorBody.message ?? message;
        } catch {
          // ignore JSON parse errors
        }
        throw new Error(message);
      }

      toast.success("Advisory status updated");
      setDialogOpen(false);
      setSelectedAdvisory(null);
      await loadAdvisories();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to update advisory status.",
      );
    } finally {
      setSubmitting(false);
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
          <h1 className="mb-4 text-2xl font-semibold">Security Advisories</h1>
          <div className="rounded-md border border-[#d0d7de] bg-white p-6">
            <p className="text-sm font-semibold text-[#cf222e]">Access Denied</p>
            <p className="mt-2 text-sm text-[#656d76]">
              You do not have permission to view security advisories. Admin access
              is required.
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
          / Advisories
        </div>
        <h1 className="mb-6 text-2xl font-semibold">Security Advisories</h1>

        <form
          className="mb-4 grid gap-4 rounded-md border border-[#d0d7de] bg-white p-4 sm:grid-cols-3"
          onSubmit={(event) => {
            event.preventDefault();
            loadAdvisories();
          }}
        >
          <div className="space-y-2">
            <Label htmlFor="state-filter">State</Label>
            <select
              id="state-filter"
              value={stateFilter}
              onChange={(event) => setStateFilter(event.target.value)}
              className="flex h-10 w-full rounded-md border border-[#d0d7de] bg-white px-3 py-2 text-sm"
            >
              {STATE_OPTIONS.map((option) => (
                <option key={option.value || "all"} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="severity-filter">Severity</Label>
            <select
              id="severity-filter"
              value={severityFilter}
              onChange={(event) => setSeverityFilter(event.target.value)}
              className="flex h-10 w-full rounded-md border border-[#d0d7de] bg-white px-3 py-2 text-sm"
            >
              {SEVERITY_OPTIONS.map((option) => (
                <option key={option.value || "all"} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>
          <div className="flex items-end">
            <Button type="submit" disabled={loading}>
              Apply filters
            </Button>
          </div>
        </form>

        <div className="overflow-hidden rounded-md border border-[#d0d7de] bg-white">
          {loading ? (
            <p className="p-4 text-sm text-[#656d76]">Loading…</p>
          ) : error ? (
            <p className="p-4 text-sm text-[#cf222e]">{error}</p>
          ) : advisories.length === 0 ? (
            <p className="p-4 text-sm text-[#656d76]">No advisories found.</p>
          ) : (
            <TableRoot>
              <TableHeader>
                <TableRow>
                  <TableHead>GHSA ID</TableHead>
                  <TableHead>Severity</TableHead>
                  <TableHead>Summary</TableHead>
                  <TableHead>State</TableHead>
                  <TableHead>Package</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {advisories.map((advisory) => (
                  <TableRow key={advisory.id}>
                    <TableCell className="font-mono text-xs">
                      {advisory.ghsa_id}
                    </TableCell>
                    <TableCell>
                      <Badge
                        className={cn(
                          "capitalize",
                          severityBadgeClass[advisory.severity],
                        )}
                      >
                        {advisory.severity}
                      </Badge>
                    </TableCell>
                    <TableCell>{advisory.summary}</TableCell>
                    <TableCell className="capitalize">{advisory.state}</TableCell>
                    <TableCell className="font-mono text-xs">
                      {advisory.affected_package}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          setSelectedAdvisory(advisory);
                          setDialogOpen(true);
                        }}
                      >
                        Update status
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </TableRoot>
          )}
        </div>
      </div>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <h2 className="mb-1 text-lg font-semibold">Update advisory status</h2>
        {selectedAdvisory && (
          <p className="mb-4 text-sm text-[#656d76]">
            {selectedAdvisory.ghsa_id} — {selectedAdvisory.summary}
          </p>
        )}
        <AdvisoryStatusForm
          onSubmit={(state, reason) => {
            if (!submitting) {
              void handleStatusUpdate(state, reason);
            }
          }}
        />
        <div className="mt-4 flex justify-end">
          <Button
            type="button"
            variant="outline"
            onClick={() => setDialogOpen(false)}
            disabled={submitting}
          >
            Cancel
          </Button>
        </div>
      </Dialog>
    </div>
  );
}
