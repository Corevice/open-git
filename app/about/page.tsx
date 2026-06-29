import Link from "next/link";

import { BRANDING } from "@/lib/branding";

const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  process.env.NEXT_PUBLIC_API_URL ??
  "";

interface VersionData {
  version: string;
  commit: string;
  buildDate: string;
}

async function fetchVersion(): Promise<VersionData | null> {
  try {
    const response = await fetch(`${API_BASE}/api/v1/version`, {
      headers: { Accept: "application/json" },
      cache: "no-store",
    });

    if (!response.ok) {
      return null;
    }

    const body = (await response.json()) as { data?: VersionData };
    return body.data ?? null;
  } catch {
    return null;
  }
}

export default async function AboutPage() {
  const version = await fetchVersion();

  return (
    <div className="mx-auto max-w-2xl px-6 py-12">
      <h1 className="mb-8 text-3xl font-bold">{BRANDING.appName}</h1>

      <dl className="space-y-4">
        <div>
          <dt className="text-sm font-medium text-gray-500">Version</dt>
          <dd className="mt-1 text-base">{version?.version ?? "unknown"}</dd>
        </div>
        <div>
          <dt className="text-sm font-medium text-gray-500">Commit</dt>
          <dd className="mt-1 font-mono text-sm">{version?.commit ?? "unknown"}</dd>
        </div>
        <div>
          <dt className="text-sm font-medium text-gray-500">Build Date</dt>
          <dd className="mt-1 text-base">{version?.buildDate ?? "unknown"}</dd>
        </div>
        <div>
          <dt className="text-sm font-medium text-gray-500">License</dt>
          <dd className="mt-1 text-base">{BRANDING.licenseName}</dd>
        </div>
        <div>
          <dt className="text-sm font-medium text-gray-500">Source</dt>
          <dd className="mt-1">
            <Link
              href={BRANDING.sourceUrl}
              className="text-blue-600 hover:underline"
              target="_blank"
              rel="noopener noreferrer"
            >
              {BRANDING.sourceUrl}
            </Link>
          </dd>
        </div>
      </dl>
    </div>
  );
}
