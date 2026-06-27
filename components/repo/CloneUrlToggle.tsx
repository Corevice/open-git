"use client";

import { useState } from "react";

type CloneUrlToggleProps = {
  cloneUrl: string;
  sshUrl: string;
};

export default function CloneUrlToggle({ cloneUrl, sshUrl }: CloneUrlToggleProps) {
  const [activeTab, setActiveTab] = useState<"https" | "ssh">("https");

  if (cloneUrl === "" && sshUrl === "") {
    return null;
  }

  const activeUrl = activeTab === "https" ? cloneUrl : sshUrl;

  return (
    <div className="flex items-center gap-2">
      <div className="flex border border-[#d0d7de] rounded-md overflow-hidden">
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
          className={`px-2.5 py-1.5 text-xs font-medium border-l border-[#d0d7de] ${
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
        value={activeUrl}
        className="font-mono text-sm bg-[#f6f8fa] border border-[#d0d7de] px-2.5 py-1.5 rounded-md min-w-[280px]"
      />
      <button
        type="button"
        onClick={() => navigator.clipboard.writeText(activeUrl)}
        className="px-3 py-1.5 text-sm bg-[color:var(--primary)] text-white rounded-md hover:bg-[color:var(--primary-hover)]"
      >
        📋 Copy
      </button>
    </div>
  );
}
