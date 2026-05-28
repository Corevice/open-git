"use client";

import Link from "next/link";
import { useState } from "react";

type Issue = {
  id: number;
  title: string;
  labels: { text: string; color: string }[];
  meta: string;
  comments: number;
};

const issues: Issue[] = [
  {
    id: 1247,
    title: "SSR時にハイドレーションエラーが発生する",
    labels: [
      { text: "bug", color: "bg-[#d1242f]" },
      { text: "help wanted", color: "bg-[#8250df]" },
    ],
    meta: "#1247 opened 3 hours ago by tanaka-dev · 8 comments",
    comments: 8,
  },
  {
    id: 1245,
    title: "ダークモード対応の追加要望",
    labels: [
      { text: "enhancement", color: "bg-[#1f883d]" },
      { text: "good first issue", color: "bg-[#7c3aed]" },
    ],
    meta: "#1245 opened yesterday by sato_ui · 12 comments",
    comments: 12,
  },
  {
    id: 1243,
    title: "README.mdのインストール手順が古い",
    labels: [{ text: "documentation", color: "bg-[#0969da]" }],
    meta: "#1243 opened 2 days ago by suzuki_doc · 3 comments",
    comments: 3,
  },
  {
    id: 1240,
    title: "TypeScript 5.4対応のロードマップ",
    labels: [{ text: "enhancement", color: "bg-[#1f883d]" }],
    meta: "#1240 opened 4 days ago by maintainer · 24 comments",
    comments: 24,
  },
  {
    id: 1238,
    title: "useEffect内のメモリリーク",
    labels: [{ text: "bug", color: "bg-[#d1242f]" }],
    meta: "#1238 opened 5 days ago by yamada_qa · 6 comments",
    comments: 6,
  },
];

export default function IssueListPage() {
  const [search, setSearch] = useState("is:issue is:open");
  const [comment, setComment] = useState("");
  const [selectedId, setSelectedId] = useState(1247);

  const handleComment = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      {/* App Bar */}
      <header className="sticky top-0 z-50 h-16 flex items-center justify-between px-6 bg-white/85 backdrop-blur border-b border-[#d0d7de]">
        <div className="flex items-center gap-2 font-extrabold text-lg">
          <span>🐙</span>
          <strong>OctoOSS</strong>
        </div>
        <div className="flex items-center gap-4">
          <input
            className="w-[280px] px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm bg-white"
            placeholder="Search or jump to..."
          />
          <Link href="/13-pr-list" className="text-sm text-[#1f2328] hover:text-[#0969da]">
            Pull Requests
          </Link>
          <Link href="/12-issue-list" className="text-sm text-[#1f2328] hover:text-[#0969da]">
            Issues
          </Link>
          <span className="w-7 h-7 rounded-full bg-gradient-to-br from-[#0969da] to-[#8250df] text-white text-xs font-semibold inline-flex items-center justify-center">
            YK
          </span>
        </div>
      </header>

      {/* Repo Nav */}
      <nav className="bg-white border-b border-[#d0d7de] px-6 flex gap-1">
        <Link href="/07-repo-detail" className="px-4 py-3 text-sm border-b-2 border-transparent hover:border-[#d0d7de]">
          📄 Code
        </Link>
        <Link
          href="/12-issue-list"
          className="px-4 py-3 text-sm font-semibold border-b-2 border-[#fd8c73]"
        >
          ⊙ Issues <span className="ml-1 px-1.5 rounded-full text-xs bg-[#ddf4ff] text-[#0969da]">42</span>
        </Link>
        <Link href="/13-pr-list" className="px-4 py-3 text-sm border-b-2 border-transparent hover:border-[#d0d7de]">
          ⇄ Pull requests <span className="ml-1 px-1.5 rounded-full text-xs bg-[#eaeef2]">7</span>
        </Link>
        <Link href="/12-issue-list" className="px-4 py-3 text-sm border-b-2 border-transparent hover:border-[#d0d7de]">
          ▶ Actions
        </Link>
        <Link href="/12-issue-list" className="px-4 py-3 text-sm border-b-2 border-transparent hover:border-[#d0d7de]">
          📊 Insights
        </Link>
        <Link href="/12-issue-list" className="px-4 py-3 text-sm border-b-2 border-transparent hover:border-[#d0d7de]">
          ⚙ Settings
        </Link>
      </nav>

      <div className="max-w-[1280px] mx-auto px-6">
        <div className="py-4 text-xl">
          <strong className="text-[#0969da]">octocorp</strong>
          <span className="mx-1">/</span>
          <strong className="text-[#0969da]">awesome-framework</strong>
          <span className="ml-2 px-2 py-0.5 text-xs rounded-full border border-[#d0d7de] text-[#656d76]">
            Public
          </span>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-[1fr_420px] gap-5 py-5">
          {/* Issue List */}
          <div>
            <div className="flex justify-between items-center mb-3">
              <div className="flex gap-2">
                <button className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white hover:bg-[#f6f8fa]">
                  Labels <span className="ml-1 text-[#656d76]">18</span>
                </button>
                <button className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white hover:bg-[#f6f8fa]">
                  Milestones <span className="ml-1 text-[#656d76]">4</span>
                </button>
              </div>
              <Link
                href="/12-issue-list"
                className="bg-[#1f883d] text-white px-4 py-1.5 rounded-md font-semibold text-sm border border-black/10"
              >
                New issue
              </Link>
            </div>

            {/* Filter Bar */}
            <div className="flex gap-2 items-center bg-[#f6f8fa] border border-[#d0d7de] rounded-t-md px-4 py-3">
              <input type="checkbox" />
              <input
                type="text"
                className="flex-1 px-3 py-1.5 border border-[#d0d7de] rounded-md bg-white text-sm"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
              {["Author", "Label", "Projects", "Milestones", "Assignee", "Sort"].map((d) => (
                <button key={d} className="px-3 py-1.5 text-sm text-[#1f2328] hover:text-[#0969da]">
                  {d} ▾
                </button>
              ))}
            </div>

            <div className="flex gap-2 items-center bg-[#f6f8fa] border-x border-[#d0d7de] px-4 py-3">
              <div className="flex gap-4 text-sm">
                <span className="font-semibold text-[#1f2328]">⊙ 42 Open</span>
                <span className="text-[#656d76]">✓ 187 Closed</span>
              </div>
            </div>

            {/* Issue Table */}
            <div className="bg-white border border-t-0 border-[#d0d7de] rounded-b-md">
              {issues.map((issue) => {
                const selected = issue.id === selectedId;
                return (
                  <div
                    key={issue.id}
                    onClick={() => setSelectedId(issue.id)}
                    className={`flex items-start gap-3 px-4 py-3 border-t border-[#d0d7de] cursor-pointer ${
                      selected
                        ? "bg-[#ddf4ff] border-l-[3px] border-l-[#0969da] pl-[13px]"
                        : "hover:bg-[#f6f8fa]"
                    }`}
                  >
                    <input type="checkbox" className="mt-1" />
                    <span className="text-[#1a7f37] text-base mt-0.5">⊙</span>
                    <div className="flex-1 min-w-0">
                      <Link
                        href="/12-issue-list"
                        className="text-[15px] font-semibold text-[#1f2328] hover:text-[#0969da]"
                      >
                        {issue.title}
                      </Link>
                      {issue.labels.map((l) => (
                        <span
                          key={l.text}
                          className={`inline-block px-2 rounded-full text-[11px] font-semibold leading-[18px] ml-1.5 text-white ${l.color}`}
                        >
                          {l.text}
                        </span>
                      ))}
                      <div className="text-xs text-[#656d76] mt-1">{issue.meta}</div>
                    </div>
                    <div className="text-xs text-[#656d76] flex items-center gap-2 whitespace-nowrap">
                      {selected && (
                        <span className="w-5 h-5 rounded-full bg-gradient-to-br from-[#0969da] to-[#8250df] text-white text-[9px] font-semibold inline-flex items-center justify-center">
                          YK
                        </span>
                      )}
                      💬 {issue.comments}
                    </div>
                  </div>
                );
              })}
            </div>

            {/* Pagination */}
            <div className="flex justify-center gap-1 py-5">
              <Link href="/12-issue-list" className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm text-[#0969da]">
                ← Prev
              </Link>
              {[1, 2, 3, 4, 5].map((p) => (
                <Link
                  key={p}
                  href="/12-issue-list"
                  className={`px-3 py-1.5 border rounded-md text-sm ${
                    p === 1
                      ? "bg-[#0969da] text-white border-[#0969da]"
                      : "border-[#d0d7de] text-[#0969da]"
                  }`}
                >
                  {p}
                </Link>
              ))}
              <Link href="/12-issue-list" className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm text-[#0969da]">
                Next →
              </Link>
            </div>
          </div>

          {/* Detail Panel */}
          <aside className="bg-white border border-[#d0d7de] rounded-md p-5 h-fit lg:sticky lg:top-20">
            <div className="inline-flex items-center gap-1 px-3 py-1 rounded-full bg-[#1a7f37] text-white text-xs font-semibold mb-3">
              ⊙ Open
            </div>
            <h2 className="text-xl mb-2 font-bold">
              SSR時にハイドレーションエラーが発生する{" "}
              <span className="text-[#656d76] font-normal">#1247</span>
            </h2>
            <div className="text-[13px] text-[#656d76] mb-2">
              <strong>tanaka-dev</strong> opened this issue 3 hours ago · 8 comments
            </div>

            <div className="py-3 border-t border-[#d0d7de]">
              <h4 className="mb-2 text-xs text-[#656d76] font-semibold uppercase">Assignees</h4>
              <div className="flex items-center gap-2 text-[13px]">
                <span className="w-6 h-6 rounded-full bg-gradient-to-br from-[#0969da] to-[#8250df] text-white text-[11px] font-semibold inline-flex items-center justify-center">
                  YK
                </span>
                yuki-kobayashi
              </div>
              <div className="flex items-center gap-2 text-[13px] mt-1.5">
                <span className="w-6 h-6 rounded-full bg-gradient-to-br from-[#1f883d] to-[#0969da] text-white text-[11px] font-semibold inline-flex items-center justify-center">
                  MT
                </span>
                maintainer-team
              </div>
            </div>

            <div className="py-3 border-t border-[#d0d7de]">
              <h4 className="mb-2 text-xs text-[#656d76] font-semibold uppercase">Labels</h4>
              <span className="inline-block px-2 rounded-full text-[11px] font-semibold leading-[18px] text-white bg-[#d1242f] mr-1.5">
                bug
              </span>
              <span className="inline-block px-2 rounded-full text-[11px] font-semibold leading-[18px] text-white bg-[#8250df]">
                help wanted
              </span>
            </div>

            <div className="py-3 border-t border-[#d0d7de]">
              <h4 className="mb-2 text-xs text-[#656d76] font-semibold uppercase">Milestone</h4>
              <Link href="/12-issue-list" className="text-[13px] text-[#0969da]">
                📌 v2.5.0 Release (78%)
              </Link>
            </div>

            <div className="py-3 border-t border-[#d0d7de]">
              <h4 className="mb-2 text-xs text-[#656d76] font-semibold uppercase">
                Linked Pull Requests
              </h4>
              <Link href="/13-pr-list" className="text-[13px] text-[#0969da]">
                ⇄ #1251 Fix hydration on SSR
              </Link>
            </div>

            <div className="py-3 border-t border-[#d0d7de]">
              <h4 className="mb-2 text-xs text-[#656d76] font-semibold uppercase">
                Activity (3 of 8)
              </h4>

              <div className="py-3 border-t border-[#d0d7de]">
                <div className="flex items-center gap-2 text-[13px] mb-1.5">
                  <span className="w-6 h-6 rounded-full bg-gradient-to-br from-[#0969da] to-[#8250df] text-white text-[11px] font-semibold inline-flex items-center justify-center">
                    TD
                  </span>
                  <span className="font-semibold">tanaka-dev</span>
                  <span className="text-[#656d76]">commented 3 hours ago</span>
                </div>
                <div className="text-sm pl-8">
                  Next.js 14.1でSSR有効時にハイドレーションエラーが出ます。再現コードは以下:
                  <br />
                  <code className="bg-[#f6f8fa] px-1 py-0.5 rounded">
                    npm create awesome@latest -- --ssr
                  </code>
                </div>
              </div>

              <div className="py-3 border-t border-[#d0d7de]">
                <div className="flex items-center gap-2 text-[13px] mb-1.5">
                  <span className="w-6 h-6 rounded-full bg-gradient-to-br from-[#0969da] to-[#8250df] text-white text-[11px] font-semibold inline-flex items-center justify-center">
                    YK
                  </span>
                  <span className="font-semibold">yuki-kobayashi</span>
                  <span className="text-[#656d76]">commented 2 hours ago</span>
                </div>
                <div className="text-sm pl-8">
                  確認しました。<code className="bg-[#f6f8fa] px-1 py-0.5 rounded">useLayoutEffect</code>
                  がサーバーで実行されているのが原因のようです。調査中。
                </div>
              </div>

              <div className="py-3 border-t border-[#d0d7de]">
                <div className="flex items-center gap-2 text-[13px] mb-1.5">
                  <span className="w-6 h-6 rounded-full bg-gradient-to-br from-[#1f883d] to-[#0969da] text-white text-[11px] font-semibold inline-flex items-center justify-center">
                    MT
                  </span>
                  <span className="font-semibold">maintainer-team</span>
                  <span className="text-[#656d76]">commented 1 hour ago</span>
                </div>
                <div className="text-sm pl-8">
                  PR #1251 を出しました。レビューお願いします。
                </div>
              </div>
            </div>

            {/* Editor */}
            <form onSubmit={handleComment} className="mt-4">
              <div className="flex gap-1 p-1.5 border border-b-0 border-[#d0d7de] rounded-t-md bg-[#f6f8fa] text-sm">
                {["B", "I", "<>", "🔗", "📎", "@"].map((t) => (
                  <span key={t} className="px-2 py-1 cursor-pointer rounded hover:bg-[#eaeef2]">
                    {t}
                  </span>
                ))}
              </div>
              <textarea
                className="w-full min-h-[100px] p-2 border border-[#d0d7de] rounded-b-md text-sm resize-y"
                value={comment}
                onChange={(e) => setComment(e.target.value)}
                placeholder="Leave a comment"
              />
              <div className="flex gap-2 justify-end mt-3">
                <button
                  type="button"
                  className="px-4 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white hover:bg-[#f6f8fa]"
                >
                  Close issue
                </button>
                <button
                  type="submit"
                  className="px-4 py-1.5 text-sm border border-black/10 rounded-md bg-[#1f883d] text-white font-semibold"
                >
                  Comment
                </button>
              </div>
            </form>
          </aside>
        </div>
      </div>
    </div>
  );
}
