"use client";

import Link from "next/link";
import { useState } from "react";

type Run = {
  id: string;
  icon: string;
  iconColor: string;
  title: string;
  workflow: string;
  branch: string;
  trigger: string;
  sha: string;
  status: "Success" | "Failed" | "In progress" | "Cancelled";
  duration: string;
  time: string;
};

const runs: Run[] = [
  {
    id: "4521",
    icon: "✅",
    iconColor: "text-[#1a7f37]",
    title: "feat: ユーザー認証フローの改善 #4521",
    workflow: "CI Build",
    branch: "main",
    trigger: "push by yamada-taro",
    sha: "a3f9d21",
    status: "Success",
    duration: "2m 14s",
    time: "5分前",
  },
  {
    id: "4520",
    icon: "❌",
    iconColor: "text-[#cf222e]",
    title: "fix: DB接続プールのリーク修正 #4520",
    workflow: "Test Suite",
    branch: "develop",
    trigger: "pull_request by suzuki-hanako",
    sha: "b7c2e44",
    status: "Failed",
    duration: "4m 38s",
    time: "23分前",
  },
  {
    id: "4519",
    icon: "🟡",
    iconColor: "text-[#9a6700]",
    title: "chore: 依存パッケージの更新 #4519",
    workflow: "Deploy Production",
    branch: "main",
    trigger: "workflow_dispatch by tanaka-jiro",
    sha: "d1e8a09",
    status: "In progress",
    duration: "1m 02s",
    time: "1分前",
  },
  {
    id: "4518",
    icon: "✅",
    iconColor: "text-[#1a7f37]",
    title: "docs: READMEのセットアップ手順を追記 #4518",
    workflow: "Lint & Format",
    branch: "feature/docs",
    trigger: "push by yamada-taro",
    sha: "f0a3b71",
    status: "Success",
    duration: "0m 47s",
    time: "42分前",
  },
  {
    id: "4517",
    icon: "⊘",
    iconColor: "text-[#656d76]",
    title: "refactor: APIレスポンス型の整理 #4517",
    workflow: "CI Build",
    branch: "refactor/api",
    trigger: "push by sato-ichiro",
    sha: "9b4e6f2",
    status: "Cancelled",
    duration: "0m 33s",
    time: "1時間前",
  },
];

const workflows = [
  { name: "▶ All workflows", active: true },
  { name: "🔧 CI Build", active: false },
  { name: "🧪 Test Suite", active: false },
  { name: "🚀 Deploy Production", active: false },
  { name: "📦 Release Package", active: false },
  { name: "🔍 Lint & Format", active: false },
  { name: "🛡 CodeQL Analysis", active: false },
];

function statusBadge(status: Run["status"]) {
  switch (status) {
    case "Success":
      return "bg-[#dafbe1] text-[#1a7f37]";
    case "Failed":
      return "bg-[#ffebe9] text-[#cf222e]";
    case "In progress":
      return "bg-[#fff8c5] text-[#9a6700]";
    case "Cancelled":
      return "bg-[#eaeef2] text-[#656d76]";
  }
}

export default function Page() {
  const [filter, setFilter] = useState("");
  const [event, setEvent] = useState("");
  const [status, setStatus] = useState("");
  const [branch, setBranch] = useState("");
  const [actor, setActor] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <header className="sticky top-0 z-50 h-16 flex items-center justify-between px-6 bg-white/85 backdrop-blur border-b border-[var(--border)]">
        <div className="flex items-center gap-3 font-extrabold text-lg">
          <span>🐙</span>
          <strong>OpenHub</strong>
        </div>
        <div className="flex items-center gap-4">
          <Link
            href="/07-repo-detail"
            className="px-3 py-1.5 text-sm rounded-md border border-[#d0d7de] hover:bg-[#f3f4f6]"
          >
            ← リポジトリへ戻る
          </Link>
        </div>
      </header>

      <div className="bg-white border-b border-[#d0d7de] px-6 py-4">
        <div className="text-xl font-semibold">
          <Link href="/07-repo-detail" className="text-[#0969da] hover:underline">
            openhub
          </Link>
          {" / "}
          <Link href="/07-repo-detail" className="text-[#0969da] hover:underline">
            <strong>awesome-project</strong>
          </Link>
          <span className="ml-2 inline-block px-2 py-0.5 text-xs rounded-full border border-[#d0d7de] text-[#656d76] align-middle">
            Public
          </span>
        </div>
        <nav className="flex gap-1 mt-4">
          <Link href="/07-repo-detail" className="px-4 py-2 text-sm rounded-t-md flex items-center gap-1.5 hover:bg-[#f3f4f6]">
            📄 Code
          </Link>
          <Link href="/08-issues-list" className="px-4 py-2 text-sm rounded-t-md flex items-center gap-1.5 hover:bg-[#f3f4f6]">
            ⊙ Issues <span className="text-xs text-[#656d76]">23</span>
          </Link>
          <Link href="/09-pr-list" className="px-4 py-2 text-sm rounded-t-md flex items-center gap-1.5 hover:bg-[#f3f4f6]">
            ⇆ Pull requests <span className="text-xs text-[#656d76]">5</span>
          </Link>
          <Link
            href="/10-actions-list"
            className="px-4 py-2 text-sm rounded-t-md flex items-center gap-1.5 border-b-2 border-[#fd8c73] font-semibold"
          >
            ▶ Actions
          </Link>
          <Link href="/07-repo-detail" className="px-4 py-2 text-sm rounded-t-md flex items-center gap-1.5 hover:bg-[#f3f4f6]">
            📊 Insights
          </Link>
          <Link href="/07-repo-detail" className="px-4 py-2 text-sm rounded-t-md flex items-center gap-1.5 hover:bg-[#f3f4f6]">
            ⚙ Settings
          </Link>
        </nav>
      </div>

      <div className="max-w-[1280px] mx-auto px-6 py-6">
        <div className="flex gap-4 mb-4">
          <div className="flex-1 bg-white border border-[#d0d7de] rounded-md px-4 py-3">
            <div className="text-2xl font-semibold text-[#1a7f37]">142</div>
            <div className="text-xs text-[#656d76]">Success</div>
          </div>
          <div className="flex-1 bg-white border border-[#d0d7de] rounded-md px-4 py-3">
            <div className="text-2xl font-semibold text-[#cf222e]">8</div>
            <div className="text-xs text-[#656d76]">Failed</div>
          </div>
          <div className="flex-1 bg-white border border-[#d0d7de] rounded-md px-4 py-3">
            <div className="text-2xl font-semibold text-[#9a6700]">3</div>
            <div className="text-xs text-[#656d76]">In progress</div>
          </div>
          <div className="flex-1 bg-white border border-[#d0d7de] rounded-md px-4 py-3">
            <div className="text-2xl font-semibold text-[#656d76]">12</div>
            <div className="text-xs text-[#656d76]">Cancelled</div>
          </div>
        </div>

        <div className="grid grid-cols-[256px_1fr] gap-6">
          <aside>
            <div className="text-xs uppercase text-[#656d76] mb-2">Workflows</div>
            {workflows.map((w) => (
              <Link
                key={w.name}
                href="/10-actions-list"
                className={`block px-3 py-2 rounded-md text-sm mb-0.5 ${
                  w.active
                    ? "bg-[#ddf4ff] text-[#0969da] font-semibold"
                    : "text-[#1f2328] hover:bg-[#f3f4f6]"
                }`}
              >
                {w.name}
              </Link>
            ))}

            <div className="text-xs uppercase text-[#656d76] mb-2 mt-6">Management</div>
            <Link href="/10-actions-list" className="block px-3 py-2 rounded-md text-sm mb-0.5 hover:bg-[#f3f4f6]">
              ⚙ Caches
            </Link>
            <Link href="/10-actions-list" className="block px-3 py-2 rounded-md text-sm mb-0.5 hover:bg-[#f3f4f6]">
              ⚙ Runners
            </Link>
            <Link href="/10-actions-list" className="block px-3 py-2 rounded-md text-sm mb-0.5 hover:bg-[#f3f4f6]">
              ⚙ Attestations
            </Link>
          </aside>

          <main>
            <form onSubmit={handleSubmit} className="flex gap-2 items-center mb-4 flex-wrap">
              <input
                type="text"
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                placeholder="🔍 Filter workflow runs (例: branch:main event:push)"
                className="flex-1 min-w-[240px] px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md"
              />
              <select
                value={event}
                onChange={(e) => setEvent(e.target.value)}
                className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-[#f6f8fa]"
              >
                <option value="">Event ▾</option>
                <option>push</option>
                <option>pull_request</option>
                <option>workflow_dispatch</option>
                <option>schedule</option>
              </select>
              <select
                value={status}
                onChange={(e) => setStatus(e.target.value)}
                className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-[#f6f8fa]"
              >
                <option value="">Status ▾</option>
                <option>Success</option>
                <option>Failure</option>
                <option>In progress</option>
                <option>Cancelled</option>
              </select>
              <select
                value={branch}
                onChange={(e) => setBranch(e.target.value)}
                className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-[#f6f8fa]"
              >
                <option value="">Branch ▾</option>
                <option>main</option>
                <option>develop</option>
                <option>feature/*</option>
              </select>
              <select
                value={actor}
                onChange={(e) => setActor(e.target.value)}
                className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-[#f6f8fa]"
              >
                <option value="">Actor ▾</option>
                <option>yamada-taro</option>
                <option>suzuki-hanako</option>
              </select>
              <Link
                href="/10-actions-list"
                className="px-3 py-1.5 text-sm rounded-md border border-[#d0d7de] hover:bg-[#f3f4f6]"
              >
                ↕ Sort
              </Link>
            </form>

            <div className="bg-white border border-[#d0d7de] rounded-md overflow-hidden">
              <div className="px-4 py-3 bg-[#f6f8fa] border-b border-[#d0d7de] text-sm text-[#656d76] flex justify-between items-center">
                <span>
                  <strong>168</strong> workflow runs
                </span>
                <span className="text-[#94a3b8]">最終更新: 2分前</span>
              </div>

              {runs.map((run, idx) => (
                <div
                  key={run.id}
                  className={`grid grid-cols-[32px_1fr_auto_auto_auto_auto] gap-3 items-center px-4 py-3 border-b border-[#d8dee4] last:border-b-0 ${
                    idx % 2 === 1 ? "bg-[#fafbfc]" : ""
                  }`}
                >
                  <span className={`text-base ${run.iconColor}`} title={run.status}>
                    {run.icon}
                  </span>
                  <div>
                    <div className="text-sm font-semibold">
                      <Link href="/10-actions-list" className="text-[#1f2328] hover:text-[#0969da]">
                        {run.title}
                      </Link>
                    </div>
                    <div className="text-xs text-[#656d76] mt-0.5 flex gap-2 items-center flex-wrap">
                      <span>{run.workflow}</span>
                      <span>·</span>
                      <span className="bg-[#ddf4ff] text-[#0969da] px-1.5 py-0.5 rounded font-mono text-[11px]">
                        {run.branch}
                      </span>
                      <span>·</span>
                      <span>{run.trigger}</span>
                      <span>·</span>
                      <span className="font-mono">{run.sha}</span>
                    </div>
                  </div>
                  <span className={`px-2 py-0.5 rounded-full text-xs ${statusBadge(run.status)}`}>
                    {run.status}
                  </span>
                  <span className="text-xs text-[#656d76] min-w-[70px] text-right">{run.duration}</span>
                  <span className="text-xs text-[#656d76] min-w-[100px] text-right">{run.time}</span>
                  <div className="flex gap-1">
                    <Link
                      href="/10-actions-list"
                      className="px-2 py-1 text-xs border border-[#d0d7de] rounded-md hover:bg-[#f3f4f6]"
                      title="ログ"
                    >
                      📄
                    </Link>
                    <Link
                      href="/10-actions-list"
                      className="px-2 py-1 text-xs border border-[#d0d7de] rounded-md hover:bg-[#f3f4f6]"
                      title={run.status === "In progress" ? "キャンセル" : "再実行"}
                    >
                      {run.status === "In progress" ? "✕" : "↻"}
                    </Link>
                  </div>
                </div>
              ))}
            </div>

            <div className="mt-6 rounded-t-md bg-[#161b22] text-[#e6edf3] px-4 py-2 flex justify-between items-center text-sm">
              <span>📋 build (ubuntu-latest) · CI Build #4521</span>
              <span className="text-[#7d8590]">2m 14s</span>
            </div>
            <div className="bg-[#0d1117] text-[#e6edf3] rounded-b-md p-4 font-mono text-xs leading-relaxed max-h-80 overflow-y-auto">
              <div className="whitespace-pre">
                <span className="text-[#7d8590] mr-2">10:23:01</span>
                <span>Set up job</span>
              </div>
              <div className="whitespace-pre">
                <span className="text-[#7d8590] mr-2">10:23:02</span>
                <span>Checkout repository</span>
              </div>
              <div className="whitespace-pre text-[#3fb950]">
                <span className="text-[#7d8590] mr-2">10:23:05</span>
                <span>✓ Dependencies installed (847 packages)</span>
              </div>
              <div className="whitespace-pre text-[#d29922]">
                <span className="text-[#7d8590] mr-2">10:23:48</span>
                <span>⚠ 2 deprecation warnings detected</span>
              </div>
              <div className="whitespace-pre text-[#3fb950]">
                <span className="text-[#7d8590] mr-2">10:25:15</span>
                <span>✓ Build completed successfully</span>
              </div>
            </div>

            <div className="flex justify-center gap-1 mt-6">
              <Link href="/10-actions-list" className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md hover:bg-[#f3f4f6]">
                ←
              </Link>
              <Link
                href="/10-actions-list"
                className="px-3 py-1.5 text-sm border border-[#0969da] bg-[#0969da] text-white rounded-md"
              >
                1
              </Link>
              <Link href="/10-actions-list" className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md hover:bg-[#f3f4f6]">
                2
              </Link>
              <Link href="/10-actions-list" className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md hover:bg-[#f3f4f6]">
                3
              </Link>
              <Link href="/10-actions-list" className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md hover:bg-[#f3f4f6]">
                →
              </Link>
            </div>
          </main>
        </div>
      </div>
    </div>
  );
}
