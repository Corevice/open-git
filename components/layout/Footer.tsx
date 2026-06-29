"use client";

import { BRANDING } from "@/lib/branding";
import { env } from "@/lib/env";

export function Footer() {
  const version = env.NEXT_PUBLIC_APP_VERSION || "dev";

  return (
    <footer className="flex items-center gap-6 bg-[#24292f] px-6 py-4 text-sm text-gray-300">
      <span>{BRANDING.licenseName}</span>
      <a href={BRANDING.sourceUrl}>Source code</a>
      <span>{version}</span>
    </footer>
  );
}
