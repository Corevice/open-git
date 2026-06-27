"use client";

import { useMemo, useState } from "react";

interface Props {
  owner: string;
  repo: string;
}

export default function CloneUrlCopy({ owner, repo }: Props) {
  const [activeTab, setActiveTab] = useState<"https" | "ssh">("https");

  const httpsUrl = `${process.env.NEXT_PUBLIC_API_URL}/${owner}/${repo}.git`;
  const sshUrl =
    typeof window !== "undefined"
      ? `git@${window.location.hostname}:${owner}/${repo}.git`
      : `git@localhost:${owner}/${repo}.git`;

  const url = useMemo(
    () => (activeTab === "https" ? httpsUrl : sshUrl),
    [activeTab, httpsUrl, sshUrl],
  );

  return (
    <div className="flex items-center gap-2">
      <div className="flex overflow-hidden rounded-md border border-[#d0d7de]">
        <button
          type="button"
          onClick={() => setActiveTab("https")}
          className={`px-2.5 py-1.5 text-xs font-medium ${
            activeTab === "https"
              ? "bg-[color:var(--primary)] text-white"
              : "bg-white text-[#24292f] hover:bg-gray-50"
          }`}
        >
          HTTPS
        </button>
        <button
          type="button"
          onClick={() => setActiveTab("ssh")}
          className={`border-l border-[#d0d7de] px-2.5 py-1.5 text-xs font-medium ${
            activeTab === "ssh"
              ? "bg-[color:var(--primary)] text-white"
              : "bg-white text-[#24292f] hover:bg-gray-50"
          }`}
        >
          SSH
        </button>
      </div>
      <input
        readOnly
        value={url}
        className="min-w-[280px] rounded-md border border-[#d0d7de] bg-[#f6f8fa] px-2.5 py-1.5 font-mono text-sm"
      />
      <button
        type="button"
        onClick={() => navigator.clipboard.writeText(url)}
        className="rounded-md bg-[color:var(--primary)] px-3 py-1.5 text-sm text-white hover:bg-[color:var(--primary-hover)]"
      >
        Copy
      </button>
    </div>
  );
}
