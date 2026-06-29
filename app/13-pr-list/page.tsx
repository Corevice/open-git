"use client";

import Link from "next/link";
import { useState } from "react";

const prList = [
  {
    id: 247,
    title: "Add dark mode support to settings page",
    author: "alice-dev",
    meta: "opened 2 hours ago",
    checks: "✓ 12 checks passed · 💬 5",
    state: "ready",
    selected: true,
  },
  {
    id: 246,
    title: "Refactor authentication middleware",
    author: "bob-coder",
    meta: "opened yesterday",
    checks: "⚠ 1 check failing · 💬 12",
    state: "open",
  },
  {
    id: 245,
    title: "[WIP] Migrate to TypeScript 5.0",
    author: "charlie",
    meta: "opened 3 days ago",
    checks: "💬 3",
    state: "draft",
  },
  {
    id: 244,
    title: "Fix memory leak in WebSocket handler",
    author: "diana",
    meta: "opened 5 days ago",
    checks: "💬 8",
    state: "conflicts",
  },
];

export default function Page() {
  const [comment, setComment] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <header className="h-16 bg-white/85 backdrop-blur border-b border-[#d0d7de] flex items-center justify-between px-6 sticky top-0 z-[100]">
        <div className="font-extrabold text-lg flex items-center gap-2">
          <span>🐙</span>
          <span>OpenHub</span>
        </div>
        <div className="flex items-center gap-4">
          <Link href="/12-issue-list" className="text-sm text-[#24292f] hover:text-[#0969da]">Issues</Link>
          <Link href="/13-pr-list" className="text-sm text-[#24292f] hover:text-[#0969da]">Pull Requests</Link>
          <span className="w-6 h-6 rounded-full bg-gradient-to-br from-[#fd8c73] to-[#d4a017] inline-block" />
        </div>
      </header>

      <div className="bg-white px-6 pt-5 pb-3 border-b border-[#d0d7de]">
        <h1 className="m-0 text-xl font-normal">
          📦 <Link href="/07-repo-detail" className="text-[#0969da] no-underline">octocat</Link> /{" "}
          <strong className="font-semibold text-[#0969da]">hello-world</strong>{" "}
          <span className="ml-2 text-xs px-2 py-0.5 rounded-full bg-[#ddf4ff] text-[#0969da] border border-[#0969da]/20">Public</span>
        </h1>
      </div>

      <nav className="bg-white border-b border-[#d0d7de] px-6 flex gap-1">
        <Link href="/07-repo-detail" className="px-4 py-3.5 text-sm text-[#24292f] border-b-2 border-transparent hover:bg-[#f6f8fa]">&lt;&gt; Code</Link>
        <Link href="/12-issue-list" className="px-4 py-3.5 text-sm text-[#24292f] border-b-2 border-transparent hover:bg-[#f6f8fa]">⊙ Issues <span className="ml-1 text-xs bg-[#eaeef2] px-1.5 py-0.5 rounded-full">42</span></Link>
        <Link href="/13-pr-list" className="px-4 py-3.5 text-sm text-[#24292f] border-b-2 border-[#fd8c73] font-semibold">⇆ Pull Requests <span className="ml-1 text-xs bg-[#eaeef2] px-1.5 py-0.5 rounded-full">8</span></Link>
        <Link href="/13-pr-list" className="px-4 py-3.5 text-sm text-[#24292f] border-b-2 border-transparent hover:bg-[#f6f8fa]">▶ Actions</Link>
        <Link href="/13-pr-list" className="px-4 py-3.5 text-sm text-[#24292f] border-b-2 border-transparent hover:bg-[#f6f8fa]">📊 Insights</Link>
      </nav>

      <div className="max-w-[1280px] mx-auto px-6">
        {/* PR List Filter */}
        <div className="bg-[#f6f8fa] border border-[#d0d7de] rounded-t-lg px-4 py-3 flex gap-4 text-sm mt-6">
          <Link href="/13-pr-list" className="text-[#24292f] font-semibold">🟢 6 Open</Link>
          <Link href="/13-pr-list" className="text-[#57606a]">✓ 124 Closed</Link>
          <span className="ml-auto text-[#57606a]">Sort: Newest ▾</span>
        </div>
        <div className="bg-white border border-t-0 border-[#d0d7de] rounded-b-lg mb-6">
          {prList.map((pr) => (
            <div
              key={pr.id}
              className={`px-4 py-3 border-b border-[#d0d7de] last:border-b-0 flex gap-3 items-start ${pr.selected ? "bg-[#fff8c5] border-l-[3px] border-l-[#fd8c73]" : ""}`}
            >
              <span className={`text-lg ${pr.state === "draft" ? "text-[#6e7781]" : pr.state === "conflicts" ? "text-[#cf222e]" : "text-[#1f883d]"}`}>⇆</span>
              <div className="flex-1">
                <Link href="/13-pr-list" className="text-[15px] font-semibold text-[#24292f] hover:text-[#0969da] no-underline">{pr.title}</Link>
                {pr.state === "ready" && <span className="ml-2 text-xs px-2 py-0.5 rounded-full bg-[#dafbe1] text-[#1a7f37]">Ready</span>}
                {pr.state === "draft" && <span className="ml-2 text-xs px-2 py-0.5 rounded-full bg-[#eaeef2] text-[#57606a]">Draft</span>}
                {pr.state === "conflicts" && <span className="ml-2 text-xs px-2 py-0.5 rounded-full bg-[#ffebe9] text-[#cf222e]">Conflicts</span>}
                <div className="text-xs text-[#57606a] mt-1">#{pr.id} {pr.meta} by <strong>{pr.author}</strong> · {pr.checks}</div>
              </div>
              <span className="w-5 h-5 rounded-full bg-gradient-to-br from-[#fd8c73] to-[#d4a017] inline-block" />
            </div>
          ))}
        </div>

        {/* PR Header */}
        <div className="bg-white border border-[#d0d7de] rounded-lg p-5 mb-4">
          <div className="flex justify-between items-start gap-4 mb-3">
            <div>
              <h2 className="m-0 text-[22px] font-semibold">
                Add dark mode support to settings page <span className="text-[#6e7781] font-normal">#247</span>
              </h2>
              <div className="flex items-center gap-2 flex-wrap text-[13px] text-[#57606a] mt-2">
                <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-[#1f883d] text-white text-[13px] font-medium">🟢 Open</span>
                <span>
                  <strong>alice-dev</strong> wants to merge <strong>3 commits</strong> into{" "}
                  <code className="mono bg-[#f6f8fa] px-1.5 py-0.5 rounded">main</code> from{" "}
                  <code className="mono bg-[#f6f8fa] px-1.5 py-0.5 rounded">feature/dark-mode</code>
                </span>
              </div>
            </div>
            <div className="flex gap-2">
              <Link href="/13-pr-list" className="text-sm px-3 py-1.5 border border-[#d0d7de] rounded-md bg-white text-[#24292f] hover:bg-[#f6f8fa]">✏ Edit</Link>
              <Link href="/13-pr-list" className="text-sm px-3 py-1.5 border border-[#d0d7de] rounded-md bg-white text-[#24292f] hover:bg-[#f6f8fa]">🔗 Share</Link>
              <Link href="/13-pr-list" className="text-sm px-3 py-1.5 border border-[#cf222e] rounded-md bg-white text-[#cf222e] hover:bg-[#ffebe9]">✕ Close</Link>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-6 pb-6">
          <div>
            {/* Tabs */}
            <div className="flex gap-0 bg-white border border-b-0 border-[#d0d7de] rounded-t-lg px-4">
              <Link href="/13-pr-list" className="px-4 py-3.5 text-sm text-[#24292f] border-b-2 border-[#fd8c73] font-semibold inline-flex items-center gap-2">💬 Conversation <span className="text-xs bg-[#eaeef2] px-1.5 py-0.5 rounded-full">5</span></Link>
              <Link href="/13-pr-list" className="px-4 py-3.5 text-sm text-[#24292f] border-b-2 border-transparent hover:text-[#0969da] inline-flex items-center gap-2">⊙ Commits <span className="text-xs bg-[#eaeef2] px-1.5 py-0.5 rounded-full">3</span></Link>
              <Link href="/13-pr-list" className="px-4 py-3.5 text-sm text-[#24292f] border-b-2 border-transparent hover:text-[#0969da] inline-flex items-center gap-2">✓ Checks <span className="text-xs bg-[#eaeef2] px-1.5 py-0.5 rounded-full">12</span></Link>
              <Link href="/13-pr-list" className="px-4 py-3.5 text-sm text-[#24292f] border-b-2 border-transparent hover:text-[#0969da] inline-flex items-center gap-2">📄 Files changed <span className="text-xs bg-[#eaeef2] px-1.5 py-0.5 rounded-full">7</span></Link>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-b-lg p-5">
              {/* Comment */}
              <div className="border border-[#d0d7de] rounded-lg mb-4 overflow-hidden">
                <div className="bg-[#f6f8fa] px-4 py-2.5 border-b border-[#d0d7de] text-[13px] flex items-center gap-2">
                  <span className="w-5 h-5 rounded-full bg-gradient-to-br from-[#fd8c73] to-[#d4a017] inline-block" />
                  <strong className="text-[#24292f]">alice-dev</strong>
                  <span>commented 2 hours ago</span>
                </div>
                <div className="p-4 text-sm leading-relaxed">
                  <p>This PR adds full dark mode support to the user settings page. Users can now toggle between light, dark, and system preference modes.</p>
                  <p className="mt-2"><strong>Changes:</strong></p>
                  <ul className="list-disc pl-6 mt-1">
                    <li>Added theme provider context</li>
                    <li>Updated all settings components with theme-aware styles</li>
                    <li>Persisted preference to localStorage</li>
                  </ul>
                  <p className="mt-2">Closes #198</p>
                </div>
              </div>

              {/* Diff */}
              <div className="border border-[#d0d7de] rounded-lg mb-4 overflow-hidden">
                <div className="bg-[#f6f8fa] px-4 py-2.5 border-b border-[#d0d7de] mono text-[13px] flex justify-between">
                  <span>📄 src/components/Settings.tsx</span>
                  <span><span className="text-[#1a7f37]">+42</span> <span className="text-[#cf222e]">-8</span></span>
                </div>
                <table className="w-full border-collapse mono text-xs">
                  <tbody>
                    <tr className="bg-[#ddf4ff] text-[#57606a]">
                      <td colSpan={3} className="px-2.5 py-1">@@ -15,7 +15,9 @@ export function Settings() {"{}"}</td>
                    </tr>
                    <tr>
                      <td className="w-10 text-right text-[#6e7781] bg-[#f6f8fa] px-2.5">15</td>
                      <td className="w-10 text-right text-[#6e7781] bg-[#f6f8fa] px-2.5">15</td>
                      <td className="px-2.5">&nbsp;import React from {"'react'"};</td>
                    </tr>
                    <tr className="bg-[#ffebe9]">
                      <td className="w-10 text-right text-[#6e7781] bg-[#ffd7d5] px-2.5">16</td>
                      <td className="w-10 text-right text-[#6e7781] bg-[#ffd7d5] px-2.5"></td>
                      <td className="px-2.5">- import {"{ useState }"} from {"'react'"};</td>
                    </tr>
                    <tr className="bg-[#dafbe1]">
                      <td className="w-10 text-right text-[#6e7781] bg-[#ccffd8] px-2.5"></td>
                      <td className="w-10 text-right text-[#6e7781] bg-[#ccffd8] px-2.5">16</td>
                      <td className="px-2.5">+ import {"{ useState, useContext }"} from {"'react'"};</td>
                    </tr>
                    <tr className="bg-[#dafbe1]">
                      <td className="w-10 text-right text-[#6e7781] bg-[#ccffd8] px-2.5"></td>
                      <td className="w-10 text-right text-[#6e7781] bg-[#ccffd8] px-2.5">17</td>
                      <td className="px-2.5">+ import {"{ ThemeContext }"} from {"'../theme'"};</td>
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
                      <td className="px-2.5">+   const {"{ theme, setTheme }"} = useContext(ThemeContext);</td>
                    </tr>
                    <tr>
                      <td className="w-10 text-right text-[#6e7781] bg-[#f6f8fa] px-2.5">19</td>
                      <td className="w-10 text-right text-[#6e7781] bg-[#f6f8fa] px-2.5">21</td>
                      <td className="px-2.5">&nbsp;  return (</td>
                    </tr>
                  </tbody>
                </table>
              </div>

              {/* Review comment */}
              <div className="border border-[#d0d7de] rounded-lg mb-4 overflow-hidden">
                <div className="bg-[#f6f8fa] px-4 py-2.5 border-b border-[#d0d7de] text-[13px] flex items-center gap-2">
                  <span className="w-5 h-5 rounded-full bg-gradient-to-br from-[#fd8c73] to-[#d4a017] inline-block" />
                  <strong className="text-[#24292f]">bob-coder</strong>
                  <span>reviewed 1 hour ago ·</span>
                  <span className="text-xs px-2 py-0.5 rounded-full bg-[#dafbe1] text-[#1a7f37]">Approved</span>
                </div>
                <div className="p-4 text-sm leading-relaxed">
                  <p>LGTM! 🚀 Nice clean implementation. The theme context is well structured.</p>
                </div>
              </div>

              <div className="border border-[#d0d7de] rounded-lg mb-4 overflow-hidden">
                <div className="bg-[#f6f8fa] px-4 py-2.5 border-b border-[#d0d7de] text-[13px] flex items-center gap-2">
                  <span className="w-5 h-5 rounded-full bg-gradient-to-br from-[#fd8c73] to-[#d4a017] inline-block" />
                  <strong className="text-[#24292f]">charlie</strong>
                  <span>commented 30 min ago</span>
                </div>
                <div className="p-4 text-sm leading-relaxed">
                  <p>Could we also add a smooth transition animation when toggling themes? Otherwise looks great.</p>
                </div>
              </div>

              {/* Merge box */}
              <div className="border border-[#d0d7de] rounded-lg p-4 mt-4 bg-white flex items-center gap-4">
                <div className="text-3xl text-[#1f883d]">✓</div>
                <div className="flex-1">
                  <strong className="block mb-1">This branch has no conflicts with the base branch</strong>
                  <span className="text-[#94a3b8] text-sm">Merging can be performed automatically.</span>
                </div>
                <Link href="/13-pr-list" className="px-4 py-2 bg-[#1f883d] text-white rounded-md text-sm font-medium hover:bg-[#1a7f37]">Merge pull request ▾</Link>
              </div>

              {/* Review panel */}
              <form onSubmit={handleSubmit} className="bg-white border border-[#d0d7de] rounded-lg p-4 mt-4">
                <h3 className="m-0 mb-3 text-[15px] font-semibold">💬 Add a comment / review</h3>
                <textarea
                  value={comment}
                  onChange={(e) => setComment(e.target.value)}
                  placeholder="Leave a comment..."
                  className="w-full min-h-[80px] p-2.5 border border-[#d0d7de] rounded-md text-sm box-border"
                />
                <div className="flex gap-2 mt-3 flex-wrap">
                  <button type="submit" className="px-3 py-1.5 text-sm bg-[#f6f8fa] border border-[#d0d7de] rounded-md text-[#24292f] hover:bg-[#eaeef2]">Comment</button>
                  <button type="button" className="px-3 py-1.5 text-sm bg-[#1f883d] text-white rounded-md hover:bg-[#1a7f37]">✓ Approve</button>
                  <button type="button" className="px-3 py-1.5 text-sm bg-white border border-[#d0d7de] rounded-md text-[#24292f] hover:bg-[#f6f8fa]">Request changes</button>
                </div>
              </form>
            </div>
          </div>

          {/* Sidebar */}
          <aside>
            <div className="bg-white border border-[#d0d7de] rounded-lg p-4 mb-4">
              <h4 className="m-0 mb-3 text-[13px] text-[#57606a] uppercase tracking-wider">Reviewers</h4>
              <div className="flex items-center gap-2 text-sm py-1">
                <span className="w-5 h-5 rounded-full bg-gradient-to-br from-[#fd8c73] to-[#d4a017] inline-block" />
                <span>bob-coder</span>
                <span className="ml-auto text-xs text-[#1a7f37]">✓ approved</span>
              </div>
              <div className="flex items-center gap-2 text-sm py-1">
                <span className="w-5 h-5 rounded-full bg-gradient-to-br from-[#fd8c73] to-[#d4a017] inline-block" />
                <span>charlie</span>
                <span className="ml-auto text-xs text-[#57606a]">pending</span>
              </div>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-lg p-4 mb-4">
              <h4 className="m-0 mb-3 text-[13px] text-[#57606a] uppercase tracking-wider">Assignees</h4>
              <div className="flex items-center gap-2 text-sm py-1">
                <span className="w-5 h-5 rounded-full bg-gradient-to-br from-[#fd8c73] to-[#d4a017] inline-block" />
                <span>alice-dev</span>
              </div>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-lg p-4 mb-4">
              <h4 className="m-0 mb-3 text-[13px] text-[#57606a] uppercase tracking-wider">Labels</h4>
              <div className="flex gap-1.5 flex-wrap">
                <span className="text-xs px-2 py-0.5 rounded-full bg-[#ddf4ff] text-[#0969da]">enhancement</span>
                <span className="text-xs px-2 py-0.5 rounded-full bg-[#fef3c7] text-[#9a6700]">ui</span>
                <span className="text-xs px-2 py-0.5 rounded-full bg-[#dafbe1] text-[#1a7f37]">good first review</span>
              </div>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-lg p-4 mb-4">
              <h4 className="m-0 mb-3 text-[13px] text-[#57606a] uppercase tracking-wider">Checks</h4>
              <div className="text-[13px] py-1.5 flex items-center gap-2 border-b border-[#eaeef2]">
                <span className="text-[#1a7f37]">✓</span> build / test
              </div>
              <div className="text-[13px] py-1.5 flex items-center gap-2 border-b border-[#eaeef2]">
                <span className="text-[#1a7f37]">✓</span> lint
              </div>
              <div className="text-[13px] py-1.5 flex items-center gap-2">
                <span className="text-[#1a7f37]">✓</span> e2e
              </div>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-lg p-4 mb-4">
              <h4 className="m-0 mb-3 text-[13px] text-[#57606a] uppercase tracking-wider">Timeline</h4>
              <div className="text-[13px] text-[#57606a] py-2 pl-4 border-l-2 border-[#d0d7de] ml-2">
                alice-dev opened this PR
              </div>
              <div className="text-[13px] text-[#57606a] py-2 pl-4 border-l-2 border-[#d0d7de] ml-2">
                bob-coder approved
              </div>
              <div className="text-[13px] text-[#57606a] py-2 pl-4 border-l-2 border-[#d0d7de] ml-2">
                CI passed
              </div>
            </div>
          </aside>
        </div>
      </div>
    </div>
  );
}
