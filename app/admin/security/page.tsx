import Link from "next/link";
import { redirect } from "next/navigation";

import { SecuritySummaryCard } from "@/components/admin/SecuritySummaryCard";
import {
  SecurityAccessDenied,
  SecurityPageLayout,
} from "@/components/admin/SecurityPageLayout";
import type {
  AdvisorySeverity,
  ScanJob,
  SecurityAdvisory,
} from "@/lib/api-types";
import {
  buildOrgQueryString,
  getSecurityApiBase,
} from "@/lib/admin/security";
import { requireOrgAdminAccess } from "@/lib/admin/security-server";

const SEVERITIES: AdvisorySeverity[] = ["critical", "high", "medium", "low"];

const SEVERITY_LABELS: Record<AdvisorySeverity, string> = {
  critical: "Critical advisories",
  high: "High advisories",
  medium: "Medium advisories",
  low: "Low advisories",
};

function countBySeverity(
  advisories: SecurityAdvisory[],
): Record<AdvisorySeverity, number> {
  const counts: Record<AdvisorySeverity, number> = {
    critical: 0,
    high: 0,
    medium: 0,
    low: 0,
  };

  for (const advisory of advisories) {
    if (advisory.severity in counts) {
      counts[advisory.severity] += 1;
    }
  }

  return counts;
}

export default async function SecurityDashboardPage({
  searchParams,
}: {
  searchParams: Promise<{ org?: string }>;
}) {
  const { org: orgParam } = await searchParams;
  const access = await requireOrgAdminAccess(orgParam);

  if (access.status === "unauthenticated") {
    redirect("/login");
  }

  if (access.status === "access_denied") {
    return <SecurityAccessDenied title="Security Dashboard" />;
  }

  if (access.status === "error") {
    return (
      <SecurityPageLayout title="Security Dashboard">
        <div className="rounded-md border border-[#d0d7de] bg-white p-6">
          <p className="text-sm text-[#cf222e]">{access.message}</p>
        </div>
      </SecurityPageLayout>
    );
  }

  const { token, org } = access;
  const apiBase = getSecurityApiBase();
  const orgQuery = buildOrgQueryString(orgParam);

  const advisoriesResponse = await fetch(
    `${apiBase}/api/v3/orgs/${encodeURIComponent(org)}/security-advisories?per_page=100`,
    {
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${token}`,
      },
      cache: "no-store",
    },
  );

  if (advisoriesResponse.status === 401) {
    redirect("/login");
  }

  if (advisoriesResponse.status === 403) {
    return <SecurityAccessDenied title="Security Dashboard" />;
  }

  if (!advisoriesResponse.ok) {
    return (
      <SecurityPageLayout title="Security Dashboard">
        <div className="rounded-md border border-[#d0d7de] bg-white p-6">
          <p className="text-sm text-[#cf222e]">
            Unable to load security advisories. Please try again.
          </p>
        </div>
      </SecurityPageLayout>
    );
  }

  const advisories =
    (await advisoriesResponse.json()) as SecurityAdvisory[];
  const severityCounts = countBySeverity(advisories);

  const scanJobsResponse = await fetch(
    `${apiBase}/api/v3/orgs/${encodeURIComponent(org)}/scan-jobs?per_page=10`,
    {
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${token}`,
      },
      cache: "no-store",
    },
  );

  let scanJobs: ScanJob[] = [];
  if (scanJobsResponse.ok) {
    scanJobs = (await scanJobsResponse.json()) as ScanJob[];
  }

  return (
    <SecurityPageLayout
      title="Security Dashboard"
      backHref="/dashboard"
      backLabel="← Dashboard"
    >
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div className="flex gap-2 text-sm">
          <Link
            href={`/admin/security/advisories${orgQuery}`}
            className="rounded-md border border-[#d0d7de] bg-white px-3 py-1.5 hover:bg-[#f6f8fa]"
          >
            Advisories
          </Link>
          <Link
            href={`/admin/security/audit-log${orgQuery}`}
            className="rounded-md border border-[#d0d7de] bg-white px-3 py-1.5 hover:bg-[#f6f8fa]"
          >
            Audit log
          </Link>
        </div>
      </div>

      <div className="mb-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {SEVERITIES.map((severity) => (
          <SecuritySummaryCard
            key={severity}
            severity={severity}
            count={severityCounts[severity]}
            label={SEVERITY_LABELS[severity]}
          />
        ))}
      </div>

      <section>
        <h2 className="mb-4 text-lg font-semibold">Recent scan jobs</h2>
        <div className="overflow-hidden rounded-md border border-[#d0d7de] bg-white">
          {scanJobs.length === 0 ? (
            <p className="p-4 text-sm text-[#656d76]">No scan jobs found.</p>
          ) : (
            <table className="w-full table-auto text-sm">
              <thead className="border-b border-[#d0d7de] bg-[#f6f8fa] text-left text-xs uppercase text-[#656d76]">
                <tr>
                  <th className="px-4 py-2">Job ID</th>
                  <th className="px-4 py-2">Type</th>
                  <th className="px-4 py-2">Status</th>
                  <th className="px-4 py-2">Started</th>
                  <th className="px-4 py-2">Finished</th>
                </tr>
              </thead>
              <tbody>
                {scanJobs.map((job) => (
                  <tr
                    key={job.id}
                    className="border-b border-[#eaeef2] last:border-b-0"
                  >
                    <td className="px-4 py-2 font-mono text-xs">{job.id}</td>
                    <td className="px-4 py-2 capitalize">{job.type}</td>
                    <td className="px-4 py-2 capitalize">{job.status}</td>
                    <td className="px-4 py-2 text-xs text-[#656d76]">
                      {job.started_at ?? "—"}
                    </td>
                    <td className="px-4 py-2 text-xs text-[#656d76]">
                      {job.finished_at ?? "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </section>
    </SecurityPageLayout>
  );
}
