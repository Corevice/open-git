"use client";

import Link from "next/link";
import { useState } from "react";

const prList = [
  {
    id: 247,
    title: "Add dark mode support to settings page",
    author: "alice-dev",
    time: "2 hours ago",
    badge: { label: "Ready", cls: "bg-[color:var(--success-light)] text-[color:var(--success)]" },
    meta: "✓ 12 checks passed · 💬 5",
    state: "open" as const,
    selected: true,
    avatars: 2,
  },
  {
    id: 246,
    title: "Refactor authentication middleware",
    author: "bob-coder",
    time: "yesterday",
    badge: null,
    meta: "⚠ 1 check failing · 💬 12",
    state: "open" as const,
    selected: false,
    avatars: 1,
  },
  {
    id: 245,
    title: "[WIP] Migrate to TypeScript 5.0",
    author: "charlie",
    time: "3 days ago",
    badge: { label: "Draft", cls: "bg-[#eaeef2] text-[#57606a]" },
    meta: "💬 3",
    state: "draft" as const,
    selected: false,
    avatars: 1,
  },
  {
    id: 244,
    title: "Fix memory leak in WebSocket handler",
    author: "diana",
    time: "5 days ago",
    badge: { label: "Conflicts", cls: "bg-[color:var(--danger-light)] text-[color:var(--danger)]" },
    meta: "💬 8",
    state: "open" as const,
    selected: false,
    avatars: 1,
  },
];

export default function Page() {
  const [comment, setComment] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  const Avatar = ({ size = "size-5" }: { size?: string }) => (
    <span className={`${size} rounded-full inline-block bg-gradient-to-br from-[#fd8c73] to-[#d4a017]`} />
  );

  return (
    <div className="min-h-screen bg-[#f6f8fa] font-sans">
      {/* App bar */}
      <header className="h-16 bg-white/85 backdrop-blur border-b border-[#d0d7de] flex items-center justify-between px-6 sticky top-0 z-50">
        <div className="flex items-center gap-2 font-extrabold text-lg">
          <span>🐙</span>
          <span>OpenHub</span>
        </div>
        <div className="flex items-center gap-4">
          <Link href="/12-issue-list" className="text-sm text-[#24292f] hover:text-[color:var(--primary)]">Issues</Link>
          <Link href="/13-pr-list" className="text-sm text-[#24292f] hover:text-[color:var(--primary)]">Pull Requests</Link>
          <Avatar size="size-7" />
        </div>
      </header>

      {/* Repo title */}
      <div className="bg-white px-6 pt-5 pb-3 border-b border-[#d0d7de]">
        <h1 className="text-xl font-normal">
          📦 <Link href="/07-repo-detail" className="text-[#0969da] no-underline">octocat</Link> / <strong className="font-semibold text-[#0969da]">hello-world</strong>{" "}
          <span className="ml-2 text-xs px-2 py-0.5 rounded-full bg-[color:var(--info-light)] text-[color:var(--info)] align-middle">Public</span>
        </h1>
      </div>

      {/* Repo nav */}
      <nav className="bg-white border-b border-[#d0d7de] px-6 flex gap-1">
        <Link href="/07-repo-detail" className="px-4 py-3.5 text-sm text-[#24292f] hover:bg-[#f6f8fa] inline-flex items-center gap-1.5 border-b-2 border-transparent">&lt;&gt; Code</Link>
        <Link href="/12-issue-list" className="px-4 py-3.5 text-sm text-[#24292f] hover:bg-[#f6f8fa] inline-flex items-center gap-1.5 border-b-2 border-transparent">⊙ Issues <span className="text-xs bg-[#eaeef2] px-1.5 rounded-full">42</span></Link>
        <Link href="/13-pr-list" className="px-4 py-3.5 text-sm text-[#24292f] font-semibold inline-flex items-center gap-1.5 border-b-2 border-[#fd8c73]">⇆ Pull Requests <span className="text-xs bg-[#eaeef2] px-1.5 rounded-full">8</span></Link>
        <Link href="/13-pr-list" className="px-4 py-3.5 text-sm text-[#24292f] hover:bg-[#f6f8fa] inline-flex items-center gap-1.5 border-b-2 border-transparent">▶ Actions</Link>
        <Link href="/13-pr-list" className="px-4 py-3.5 text-sm text-[#24292f] hover:bg-[#f6f8fa] inline-flex items-center gap-1.5 border-b-2 border-transparent">📊 Insights</Link>
      </nav>

      <div className="max-w-[1280px] mx-auto px-6">
        {/* PR list filter */}
        <div className="bg-[#f6f8fa] border border-[#d0d7de] rounded-t-lg px-4 py-3 flex gap-4 text-sm mt-6">
          <Link href="/13-pr-list" className="text-[#24292f] font-semibold no-underline">🟢 6 Open</Link>
          <Link href="/13-pr-list" className="text-[#57606a] no-underline">✓ 124 Closed</Link>
          <span className="ml-auto text-[#57606a]">Sort: Newest ▾</span>
        </div>

        {/* PR list */}
        <div className="bg-white border border-t-0 border-[#d0d7de] rounded-b-lg mb-6">
          {prList.map((pr) => (
            <div
              key={pr.id}
              className={`px-4 py-3 border-b border-[#d0d7de] last:border-b-0 flex gap-3 items-start ${pr.selected ? "bg-[#fff8c5] border-l-[3px] border-l-[#fd8c73]" : ""}`}
            >
              <span className={`text-lg ${pr.state === "draft" ? "text-[#6e7781]" : "text-[#1f883d]"}`}>⇆</span>
              <div className="flex-1">
                <Link href="/13-pr-list" className="text-[15px] font-semibold text-[#24292f] hover:text-[#0969da] no-underline">
                  {pr.title}
                </Link>
                {pr.badge && (
                  <span className={`ml-2 text-xs px-2 py-0.5 rounded-full ${pr.badge.cls}`}>{pr.badge.label}</span>
                )}
                <div className="text-xs text-[#57606a] mt-1">
                  #{pr.id} opened {pr.time} by <strong>{pr.author}</strong> · {pr.meta}
                </div>
              </div>
              <div className="flex gap-1">
                {Array.from({ length: pr.avatars }).map((_, i) => (
                  <Avatar key={i} />
                ))}
              </div>
            </div>
          ))}
        </div>

        {/* PR header */}
        <div className="bg-white border border-[#d0d7de] rounded-lg p-5 mb-4">
          <div className="flex justify-between items-start gap-4 mb-3">
            <div>
              <h2 className="text-[22px] font-semibold">
                Add dark mode support to settings page <span className="text-[#6e7781] font-normal">#247</span>
              </h2>
              <div className="flex items-center gap-2 flex-wrap text-[13px] text-[#57606a] mt-2">
                <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-[#1f883d] text-white text-[13px] font-medium">🟢 Open</span>
                <span>
                  <strong>alice-dev</strong> wants to merge <strong>3 commits</strong> into <code className="font-mono text-xs bg-[#f6f8fa] px-1.5 py-0.5 rounded">main</code> from <code className="font-mono text-xs bg-[#f6f8fa] px-1.5 py-0.5 rounded">feature/dark-mode</code>
                </span>
              </div>
            </div>
            <div className="flex gap-2">
              <Link href="/13-pr-list" className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md hover:bg-[#f6f8fa] text-[#24292f]">✏ Edit</Link>
              <Link href="/13-pr-list" className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md hover:bg-[#f6f8fa] text-[#24292f]">🔗 Share</Link>
              <Link href="/13-pr-list" className="px-3 py-1.5 text-sm border border-[color:var(--danger)] text-[color:var(--danger)] rounded-md hover:bg-[color:var(--danger-light)]">✕ Close</Link>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-6 pb-6">
          <div>
            {/* Tabs */}
            <div className="flex bg-white border border-b-0 border-[#d0d7de] rounded-t-lg px-4">
              <Link href="/13-pr-list" className="px-4 py-3.5 text-sm text-[#24292f] font-semibold border-b-2 border-[#fd8c73] inline-flex items-center gap-2">💬 Conversation <span className="text-xs bg-[#eaeef2] px-1.5 rounded-full">5</span></Link>
              <Link href="/13-pr-list" className="px-4 py-3.5 text-sm text-[#24292f] hover:text-[#0969da] border-b-2 border-transparent inline-flex items-center gap-2">⊙ Commits <span className="text-xs bg-[#eaeef2] px-1.5 rounded-full">3</span></Link>
              <Link href="/13-pr-list" className="px-4 py-3.5 text-sm text-[#24292f] hover:text-[#0969da] border-b-2 border-transparent inline-flex items-center gap-2">✓ Checks <span className="text-xs bg-[#eaeef2] px-1.5 rounded-full">12</span></Link>
              <Link href="/13-pr-list" className="px-4 py-3.5 text-sm text-[#24292f] hover:text-[#0969da] border-b-2 border-transparent inline-flex items-center gap-2">📄 Files changed <span className="text-xs bg-[#eaeef2] px-1.5 rounded-full">7</span></Link>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-b-lg p-5">
              {/* Comment */}
              <div className="border border-[#d0d7de] rounded-lg mb-4 overflow-hidden">
                <div className="bg-[#f6f8fa] px-4 py-2.5 border-b border-[#d0d7de] text-[13px] flex items-center gap-2">
                  <Avatar /> <strong className="text-[#24292f]">alice-dev</strong> commented 2 hours ago
                </div>
                <div className="p-4 text-sm leading-relaxed space-y-2">
                  <p>This PR adds full dark mode support to the user settings page. Users can now toggle between light, dark, and system preference modes.</p>
                  <p><strong>Changes:</strong></p>
                  <ul className="list-disc pl-5 space-y-1">
                    <li>Added theme provider context</li>
                    <li>Updated all settings components with theme-aware styles</li>
                    <li>Persisted preference to localStorage</li>
                  </ul>
                  <p>Closes #198</p>
                </div>
              </div>

              {/* Diff */}
              <div className="border border-[#d0d7de] rounded-lg mb-4 overflow-hidden">
                <div className="bg-[#f6f8fa] px-4 py-2.5 border-b border-[#d0d7de] font-mono text-[13px] flex justify-between">
                  <span>📄 src/components/Settings.tsx</span>
                  <span><span className="text-[#1a7f37]">+42</span> <span className="text-[#cf222e]">-8</span></span>
                </div>
                <table className="w-full border-collapse font-mono text-xs">
                  <tbody>
                    <tr className="bg-[#ddf4ff] text-[#57606a]">
                      <td colSpan={3} className="px-2.5 py-1">@@ -15,7 +15,9 @@ export function Settings() {"{"}</td>
                    </tr>
                    <tr>
                      <td className="w-10 text-right text-[#6e7781] bg-[#f6f8fa] px-2.5">15</td>
                      <td className="w-10 text-right text-[#6e7781] bg-[#f6f8fa] px-2.5">15</td>
                      <td className="px-2.5">&nbsp;import React from &apos;react&apos;;</td>
                    </tr>
                    <tr className="bg-[#ffebe9]">
                      <td className="w-10 text-right text-[#6e7781] bg-[#ffd7d5] px-2.5">16</td>
                      <td className="w-10 text-right text-[#6e7781] bg-[#ffd7d5] px-2.5"></td>
                      <td className="px-2.5">- import {"{"} useState {"}"} from &apos;react&apos;;</td>
                    </tr>
                    <tr className="bg-[#dafbe1]">
                      <td className="w-10 text-right text-[#6e7781] bg-[#ccffd8] px-2.5"></td>
                      <td className="w-10 text-right text-[#6e7781] bg-[#ccffd8] px-2.5">16</td>
                      <td className="px-2.5">+ import {"{"} useState, useContext {"}"} from &apos;react&apos;;</td>
                    </tr>
                    <tr className="bg-[#dafbe1]">
                      <td className="w-10 text-right text-[#6e7781] bg-[#ccffd8] px-2.5"></td>
                      <td className="w-10 text-right text-[#6e7781] bg-[#ccffd8] px-2.5">17</td>
                      <td className="px-2.5">+ import {"{"} ThemeContext {"}"} from &apos;../theme&apos;;</td>
                    </tr>
                    <tr>
                      <td className="w-10 text-right text-[#6e7781] bg-[#f6f8fa] px-2.5">17</td>
                      <td className="w-10 text-right text-[#6e7781] bg-[#f6f8fa] px-2.5">18</td>
                      <td className="px-2.5">&nbsp;</td>
                    </tr>
                    <tr>
                      <td className="w-10 text-right text-[#6e7781] bg-[#f6f8fa] px-2.5">18</td>
                      <td className="w-10 text-right text-[#6e7781] bg-[#f6f8fa] px-2.5">19</td>
                      <td className="px-2.5">&nbsp;export function Settings() {"{"}</td>
                    </tr>
                    <tr className="bg-[#dafbe1]">
                      <td className="w-10 text-right text-[#6e7781] bg-[#ccffd8] px-2.5"></td>
                      <td className="w-10 text-right text-[#6e7781] bg-[#ccffd8] px-2.5">20</td>
                      <td className="px-2.5">+   const {"{"} theme, setTheme {"}"} = useContext(ThemeContext);</td>
                    </tr>
                  </tbody>
                </table>
              </div>

              {/* Review comment */}
              <div className="border border-[#d0d7de] rounded-lg mb-4 overflow-hidden">
                <div className="bg-[#f6f8fa] px-4 py-2.5 border-b border-[#d0d7de] text-[13px] flex items-center gap-2">
                  <Avatar /> <strong className="text-[#24292f]">bob-coder</strong> reviewed 1 hour ago ·{" "}
                  <span className="text-xs px-2 py-0.5 rounded-full bg-[color:var(--success-light)] text-[color:var(--success)]">Approved</span>
                </div>
                <div className="p-4 text-sm">
                  <p>LGTM! 🚀 Nice clean implementation. The theme context is well structured.</p>
                </div>
              </div>

              <div className="border border-[#d0d7de] rounded-lg mb-4 overflow-hidden">
                <div className="bg-[#f6f8fa] px-4 py-2.5 border-b border-[#d0d7de] text-[13px] flex items-center gap-2">
                  <Avatar /> <strong className="text-[#24292f]">charlie</strong> commented 30 min ago
                </div>
                <div className="p-4 text-sm">
                  <p>Could we also add a smooth transition animation when toggling themes? Otherwise looks great.</p>
                </div>
              </div>

              {/* Merge box */}
              <div className="border border-[#d0d7de] rounded-lg p-4 mt-4 bg-white flex items-center gap-4">
                <div className="text-3xl text-[#1f883d]">✓</div>
                <div className="flex-1">
                  <strong className="block mb-1">This branch has no conflicts with the base branch</strong>
                  <span className="text-[color:var(--text-muted)] text-sm">Merging can be performed automatically.</span>
                </div>
                <button className="px-4 py-2 text-sm font-semibold bg-[#1f883d] text-white rounded-md hover:bg-[#1a7f37]">
                  Merge pull request ▾
                </button>
              </div>

              {/* Review panel */}
              <form onSubmit={handleSubmit} className="bg-white border border-[#d0d7de] rounded-lg p-4 mt-4">
                <h3 className="mb-3 text-[15px] font-semibold">💬 Add a comment / review</h3>
                <textarea
                  value={comment}
                  onChange={(e) => setComment(e.target.value)}
                  placeholder="Leave a comment..."
                  className="w-full min-h-20 p-2.5 border border-[#d0d7de] rounded-md text-sm box-border"
                />
                <div className="flex gap-2 mt-3 flex-wrap">
                  <button type="submit" className="px-3 py-1.5 text-sm bg-[#f6f8fa] border border-[#d0d7de] rounded-md hover:bg-[#eaeef2]">Comment</button>
                  <button type="button" className="px-3 py-1.5 text-sm bg-[#1f883d] text-white rounded-md hover:bg-[#1a7f37]">✓ Approve</button>
                  <button type="button" className="px-3 py-1.5 text-sm bg-[color:var(--warning)] text-white rounded-md hover:bg-[#d97706]">⚠ Request changes</button>
                </div>
              </form>
            </div>
          </div>

          {/* Sidebar */}
          <aside className="space-y-4">
            <div className="bg-white border border-[#d0d7de] rounded-lg p-4">
              <h4 className="text-[13px] text-[#57606a] uppercase tracking-wider mb-3">Reviewers</h4>
              <div className="flex items-center gap-2 text-sm py-1">
                <Avatar /> <span>bob-coder</span>
                <span className="ml-auto text-xs px-2 py-0.5 rounded-full bg-[color:var(--success-light)] text-[color:var(--success)]">✓</span>
              </div>
              <div className="flex items-center gap-2 text-sm py-1">
                <Avatar /> <span>charlie</span>
              </div>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-lg p-4">
              <h4 className="text-[13px] text-[#57606a] uppercase tracking-wider mb-3">Assignees</h4>
              <div className="flex items-center gap-2 text-sm py-1">
                <Avatar /> <span>alice-dev</span>
              </div>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-lg p-4">
              <h4 className="text-[13px] text-[#57606a] uppercase tracking-wider mb-3">Labels</h4>
              <div className="flex flex-wrap gap-1.5">
                <span className="text-xs px-2 py-0.5 rounded-full bg-[color:var(--info-light)] text-[color:var(--info)]">enhancement</span>
                <span className="text-xs px-2 py-0.5 rounded-full bg-[#fff8c5] text-[#9a6700]">ui</span>
              </div>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-lg p-4">
              <h4 className="text-[13px] text-[#57606a] uppercase tracking-wider mb-3">Checks</h4>
              <div className="flex items-center gap-2 text-[13px] py-1.5 border-b border-[#eaeef2]">
                <span className="text-[#1a7f37]">✓</span> build / test
              </div>
              <div className="flex items-center gap-2 text-[13px] py-1.5 border-b border-[#eaeef2]">
                <span className="text-[#1a7f37]">✓</span> lint
              </div>
              <div className="flex items-center gap-2 text-[13px] py-1.5">
                <span className="text-[#1a7f37]">✓</span> e2e
              </div>
            </div>
          </aside>
        </div>
      </div>
    </div>
  );
}
