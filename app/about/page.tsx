"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { getAppMeta } from "@/lib/api";
import type { AppMeta } from "@/lib/api-types";
import { BRANDING } from "@/lib/branding";

export default function AboutPage() {
  const [meta, setMeta] = useState<AppMeta | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    getAppMeta(process.env.NEXT_PUBLIC_API_BASE_URL ?? "")
      .then((data) => {
        if (!cancelled) {
          setMeta(data);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, []);

  const version = loading ? "dev" : (meta?.version ?? "dev");
  const licenseName = meta?.license ?? BRANDING.licenseName;

  return (
    <div className="mx-auto max-w-2xl px-6 py-12">
      <h1 className="mb-6 text-3xl font-bold">{BRANDING.appName}</h1>

      {loading ? <p className="mb-4 text-muted-foreground">Loading...</p> : null}

      <dl className="space-y-4">
        <div>
          <dt className="text-sm font-medium text-muted-foreground">Version</dt>
          <dd>{version}</dd>
        </div>
        <div>
          <dt className="text-sm font-medium text-muted-foreground">License</dt>
          <dd>
            <Link href="/licenses" className="text-primary underline">
              {licenseName}
            </Link>
          </dd>
        </div>
        <div>
          <dt className="text-sm font-medium text-muted-foreground">Source</dt>
          <dd>
            <a
              href={BRANDING.sourceUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary underline"
            >
              {BRANDING.sourceUrl}
            </a>
          </dd>
        </div>
      </dl>
    </div>
  );
}
