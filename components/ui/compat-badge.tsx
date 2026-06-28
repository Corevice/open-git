"use client";

import type { ActionCompatibilityResult } from "@/lib/api-types";

const STATUS_STYLES: Record<ActionCompatibilityResult["status"], string> = {
  pass: "bg-[#dafbe1] text-[#1a7f37]",
  partial: "bg-[#fff8c5] text-[#9a6700]",
  fail: "bg-[#ffebe9] text-[#cf222e]",
  error: "bg-[#ffebe9] text-[#cf222e]",
  untested: "bg-[#eaeef2] text-[#656d76]",
};

function capitalizeStatus(
  status: ActionCompatibilityResult["status"],
): string {
  return status.charAt(0).toUpperCase() + status.slice(1);
}

export function CompatBadge({
  status,
}: {
  status: ActionCompatibilityResult["status"];
}) {
  return (
    <span
      className={`rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_STYLES[status]}`}
    >
      {capitalizeStatus(status)}
    </span>
  );
}
