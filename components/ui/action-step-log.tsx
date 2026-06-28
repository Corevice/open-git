"use client";

import { useState } from "react";

export function ActionStepLog({
  stepName,
  conclusion,
  log,
}: {
  stepName: string;
  conclusion: string | null;
  log: string;
}) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div>
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center gap-2 px-4 py-2 text-left text-sm"
      >
        <span className="font-medium">{stepName}</span>
        {conclusion !== null && (
          <span className="rounded-full bg-[#eaeef2] px-2 py-0.5 text-xs font-medium text-[#656d76]">
            {conclusion}
          </span>
        )}
      </button>
      {expanded && (
        <pre className="max-h-96 overflow-x-auto overflow-y-auto rounded-b bg-[#161b22] p-4 font-mono text-xs text-[#c9d1d9]">
          {log}
        </pre>
      )}
    </div>
  );
}
