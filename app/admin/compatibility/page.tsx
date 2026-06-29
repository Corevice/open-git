import Link from "next/link";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { CompatCoverageCard } from "@/components/admin/CompatCoverageCard";
import { CompatEndpointTable } from "@/components/admin/CompatEndpointTable";

import type { CompatReport } from "./types";

const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  process.env.NEXT_PUBLIC_API_URL ??
  "";

async function fetchCompatReport(token: string): Promise<Response> {
  return fetch(`${API_BASE}/api/v1/internal/compat/report`, {
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${token}`,
    },
    cache: "no-store",
  });
}

export default async function CompatibilityReportPage() {
  const cookieStore = await cookies();
  const token = cookieStore.get("authToken")?.value;

  if (!token) {
    redirect("/login");
  }

  const response = await fetchCompatReport(token);

  if (response.status === 401) {
    redirect("/login");
  }

  if (response.status === 403) {
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
            / Admin / Compatibility
          </div>
          <h1 className="mb-4 text-2xl font-semibold">API Compatibility Report</h1>
          <div className="rounded-md border border-[#d0d7de] bg-white p-6">
            <p className="text-sm font-semibold text-[#cf222e]">Access Denied</p>
            <p className="mt-2 text-sm text-[#656d76]">
              You do not have permission to view the compatibility report. Admin
              access is required.
            </p>
          </div>
        </div>
      </div>
    );
  }

  if (!response.ok) {
    throw new Error(`Failed to load compatibility report (${response.status})`);
  }

  const report = (await response.json()) as CompatReport;

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
          / Admin / Compatibility
        </div>
        <h1 className="mb-6 text-2xl font-semibold">API Compatibility Report</h1>

        <CompatCoverageCard
          coverage={report.coverage}
          generatedAt={report.generated_at}
        />
        <CompatEndpointTable endpoints={report.endpoints} />
      </div>
    </div>
  );
}
