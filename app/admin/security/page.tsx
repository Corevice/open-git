import Link from "next/link";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { SecuritySummaryCard } from "@/components/admin/SecuritySummaryCard";
import type {
  AdvisorySeverity,
  ScanJob,
  SecurityAdvisory,
} from "@/lib/api-types";

const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  process.env.NEXT_PUBLIC_API_URL ??
  "";

const SEVERITIES: AdvisorySeverity[] = ["critical", "high", "medium", "low"];

const SEVERITY_LABELS: Record<AdvisorySeverity, string> = {
  critical: "Critical advisories",
  high: "High advisories",
  medium: "Medium advisories",
  low: "Low advisories",
};

async function requireAdminRole(): Promise<string> {
  const cookieStore = await cookies();
  const token = cookieStore.get("authToken")?.value;

  if (!token) {
    redirect("/login");
  }

  return token;
}

async function resolveOrgLogin(
  token: string,
  orgParam?: string,
): Promise<string> {
  if (orgParam) {
    return orgParam;
  }

  const response = await fetch(`${API_BASE}/api/v3/user/orgs`, {
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${token}`,
    },
    cache: "no-store",
  });

  if (!response.ok) {
    throw new Error(`Failed to load organizations (${response.status})`);
  }

  const orgs = (await response.json()) as { login: string }[];
  if (orgs.length === 0) {
    throw new Error("No organization found");
  }

  return orgs[0].login;
}

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
    counts[advisory.severity] += 1;
  }

  return counts;
}

function AccessDenied() {
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
        <div className="mb-4 text-sm text-[#656d76]">
          <Link href="/dashboard" className="text-[#0969da]">
            Dashboard
          </Link>{" "}
          / Admin / Security
        </div>
        <h1 className="mb-4 text-2xl font-semibold">Security Dashboard</h1>
        <div className="rounded-md border border-[#d0d7de] bg-white p-6">
          <p className="text-sm font-semibold text-[#cf222e]">Access Denied</p>
          <p className="mt-2 text-sm text-[#656d76]">
            You do not have permission to view the security dashboard. Admin
            access is required.
          </p>
        </div>
      </div>
    </div>
  );
}

export default async function SecurityDashboardPage({
  searchParams,
}: {
  searchParams: Promise<{ org?: string }>;
}) {
  const token = await requireAdminRole();
  const { org: orgParam } = await searchParams;
  const org = await resolveOrgLogin(token, orgParam);

  const advisoriesResponse = await fetch(
    `${API_BASE}/api/v3/orgs/${encodeURIComponent(org)}/security-advisories?per_page=100`,
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
    return <AccessDenied />;
  }

  if (!advisoriesResponse.ok) {
    throw new Error(
      `Failed to load security advisories (${advisoriesResponse.status})`,
    );
  }

  const advisories =
    (await advisoriesResponse.json()) as SecurityAdvisory[];
  const severityCounts = countBySeverity(advisories);

  const scanJobsResponse = await fetch(
    `${API_BASE}/api/v3/orgs/${encodeURIComponent(org)}/scan-jobs?per_page=10`,
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
          href="/dashboard"
          className="rounded-md px-3 py-1.5 text-sm hover:bg-[#f6f8fa]"
        >
          ← Dashboard
        </Link>
      </header>

      <div className="mx-auto max-w-[1200px] px-6 py-6">
        <div className="mb-4 text-sm text-[#656d76]">
          <Link href="/dashboard" className="text-[#0969da]">
            Dashboard
          </Link>{" "}
          / Admin / Security
        </div>
        <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
          <h1 className="text-2xl font-semibold">Security Dashboard</h1>
          <div className="flex gap-2 text-sm">
            <Link
              href={`/admin/security/advisories${orgParam ? `?org=${encodeURIComponent(orgParam)}` : ""}`}
              className="rounded-md border border-[#d0d7de] bg-white px-3 py-1.5 hover:bg-[#f6f8fa]"
            >
              Advisories
            </Link>
            <Link
              href={`/admin/security/audit-log${orgParam ? `?org=${encodeURIComponent(orgParam)}` : ""}`}
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
      </div>
    </div>
  );
}
