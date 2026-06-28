import { listRunners } from "@/lib/api/runners";
import type { Runner } from "@/types/runner";
import { RunnersPageClient } from "./RegistrationTokenModal";

export default async function RunnersPage({
  params,
}: {
  params: Promise<{ owner: string }>;
}) {
  const { owner } = await params;

  let runners: Runner[] = [];
  let error: string | null = null;

  try {
    const data = await listRunners(owner);
    runners = data.runners;
  } catch (err) {
    error = err instanceof Error ? err.message : "Failed to load runners.";
  }

  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <div className="mx-auto max-w-[1200px] px-6 py-6">
        <h1 className="mb-6 text-2xl font-semibold">Runners</h1>

        {error ? (
          <p className="mb-4 text-sm text-[#cf222e]">{error}</p>
        ) : null}

        <RunnersPageClient owner={owner} initialRunners={runners} />
      </div>
    </div>
  );
}
