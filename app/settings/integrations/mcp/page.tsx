import { cookies } from "next/headers";
import { revalidatePath } from "next/cache";

import { CompatibilityCheckTable } from "@/components/mcp/CompatibilityCheckTable";
import { RunVerificationButton } from "@/components/mcp/RunVerificationButton";
import { Badge } from "@/components/ui/badge";
import {
  apiClient,
  type MCPLatestResult,
  type MCPVerificationRun,
} from "@/lib/api-client";
import { cn } from "@/lib/utils";

async function handleVerificationComplete() {
  "use server";
  revalidatePath("/settings/integrations/mcp");
}

function formatDate(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return iso;
  }
  return date.toLocaleString(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  });
}

function historyStatusClass(
  status: MCPVerificationRun["overall_status"],
): string {
  switch (status) {
    case "compatible":
      return "bg-[#dafbe1] text-[#1a7f37] border-[#4ac26b]";
    case "partial":
      return "bg-[#fff8c5] text-[#9a6700] border-[#d4a72c]";
    case "incompatible":
      return "bg-[#ffebe9] text-[#cf222e] border-[#ff8182]";
    default:
      return "bg-[#f6f8fa] text-[#656d76] border-[#d0d7de]";
  }
}

export default async function MCPVerificationPage() {
  const cookieStore = await cookies();
  const token = cookieStore.get("authToken")?.value;

  let latestResult: MCPLatestResult | null = null;
  let history: MCPVerificationRun[] = [];

  if (token) {
    try {
      latestResult = await apiClient.getMCPLatest({ token });
    } catch {
      latestResult = null;
    }

    try {
      history = await apiClient.getMCPHistory({ page: 1, per_page: 5, token });
    } catch {
      history = [];
    }
  }

  return (
    <main className="mx-auto max-w-[960px] px-6 py-8">
      <div className="mb-6 border-b border-[#d1d9e0] pb-4">
        <h1 className="text-2xl font-semibold">MCP Connection Verification</h1>
        <p className="mt-2 text-sm text-[#59636e]">
          Verify that MCP clients can connect to GraphQL, REST, and auth endpoints.
        </p>
      </div>

      <section className="mb-8">
        <RunVerificationButton onComplete={handleVerificationComplete} />
      </section>

      <section className="mb-8">
        {latestResult ? (
          <CompatibilityCheckTable
            checks={latestResult.checks}
            overallStatus={latestResult.overall_status}
          />
        ) : (
          <p className="text-sm text-[#59636e]">No verification runs yet</p>
        )}
      </section>

      <section>
        <h2 className="mb-4 text-lg font-semibold">Recent Runs</h2>
        {history.length === 0 ? (
          <p className="text-sm text-[#59636e]">No history available.</p>
        ) : (
          <ul className="space-y-3">
            {history.map((run) => (
              <li
                key={run.run_id}
                className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-[#d1d9e0] bg-white px-4 py-3"
              >
                <span className="font-mono text-sm">{run.repository}</span>
                <div className="flex items-center gap-3">
                  <Badge
                    variant="outline"
                    className={cn(
                      "capitalize",
                      historyStatusClass(run.overall_status),
                    )}
                  >
                    {run.overall_status ?? run.status ?? "unknown"}
                  </Badge>
                  <span className="text-sm text-[#59636e]">
                    {formatDate(run.executed_at)}
                  </span>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </main>
  );
}
