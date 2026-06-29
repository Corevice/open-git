"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  process.env.NEXT_PUBLIC_API_URL ??
  "";

function buildTimeVersion(): string | null {
  return (
    process.env.NEXT_PUBLIC_VERSION ??
    process.env.NEXT_PUBLIC_APP_VERSION ??
    null
  );
}

export function Footer() {
  const [version, setVersion] = useState<string | null>(buildTimeVersion());

  useEffect(() => {
    if (version) {
      return;
    }

    let cancelled = false;

    async function loadVersion() {
      try {
        const response = await fetch(`${API_BASE}/api/v1/version`, {
          headers: { Accept: "application/json" },
        });

        if (!response.ok) {
          return;
        }

        const body = (await response.json()) as { data?: { version?: string } };
        const fetched = body.data?.version;
        if (!cancelled && fetched) {
          setVersion(fetched);
        }
      } catch {
        // ignore fetch errors; version remains hidden
      }
    }

    void loadVersion();

    return () => {
      cancelled = true;
    };
  }, [version]);

  return (
    <footer className="border-t border-gray-200 px-6 py-4 text-sm text-gray-600">
      <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-2">
        <span>© 2025 Forge</span>
        <div className="flex items-center gap-4">
          <Link href="/about" className="hover:underline">
            About
          </Link>
          {version && <span>v{version}</span>}
        </div>
      </div>
    </footer>
  );
}
