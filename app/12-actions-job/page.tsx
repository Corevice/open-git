"use client";

import { Suspense } from "react";

import { JobLogsPageContent } from "./JobLogsPageContent";

export default function Page() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen bg-[#f6f8fa] px-6 py-8 text-[#656d76]">
          Loading job logs…
        </div>
      }
    >
      <JobLogsPageContent />
    </Suspense>
  );
}
