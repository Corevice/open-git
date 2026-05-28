"use client";

import Link from "next/link";
import { useState, FormEvent } from "react";

const runs = [
  {
    id: 4521,
    icon: "✅",
    iconColor: "text-[#1a7f37]",
    title: "feat: ユーザー認証フローの改善 #4521",
    workflow: "CI Build",
    branch: "main",
    event: "push",
    actor: "yamada-taro",
    sha: "a3f9d21",
    status: "Success",
    statusClass: "bg-[var(--success-light)] text-[var(--success)]",
    duration: "2m 14s",
    time: "5分前",
    action: "↻",
  },
  {
    id: 4520,
    icon: "❌",
    iconColor: "text-[#cf222e]",
    title: "fix: DB接続プールのリーク修正 #4520",
    workflow: "Test Suite",
    branch: "develop",
    event: "pull_request",
    actor: "suzuki-hanako",
    sha: "b7c2e44",
    status: "Failed",
    statusClass: "bg-[var(--danger-light)] text-[var(--danger)]",
    duration: "4m 38s",
    time: "23分前",
    action: "↻",
  },
  {
    id: 4519,
    icon: "🟡",
    iconColor: "text-[#9a6700]",
    title: "chore: 依存パッケージの更新 #4519",
    workflow: "Deploy Production",
    branch: "main",
    event: "workflow_dispatch",
    actor: "tanaka-jiro",
    sha: "d1e8a09",
    status: "In progress",
    statusClass: "bg-[var(--warning-light)] text-[var(--warning)]",
    duration: "1m 02s",
    time: "1分前",
    action: "✕",
  },
  {
    id: 4518,
    icon: "✅",
    iconColor: "text-[#1a7f37]",
    title: "docs: READMEのセットアップ手順を追記 #4518",
    workflow: "Lint & Format",
    branch: "feature/docs",
    event: "push",
    actor: "yamada-taro",
    sha: "f0a3b71",
    status: "Success",
    statusClass: "bg-[var(--success-light)] text-[var(--success)]",
    duration: "0m 47s",
    time: "42分前",
    action: "↻",
  },
  {
    id: 4517,
    icon: "⊘",
    iconColor: "text-[#656d76]",
    title: "refactor: APIレスポンス型の整理 #4517",
    workflow: "CI Build",
    branch: "refactor/api",
    event: "push",
    actor: "sato-ichiro",
    sha: "9b4e6f2",
    status: "Cancelled",
    statusClass: "bg-[var(--bg-muted)] text-[var(--text-secondary)]",
    duration: "0m 33s",
    time: "1時間前",
    action: "↻",
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

export default function ActionsListPage() {
  const [filter, setFilter] = useState("");
  const [event, setEvent] = useState("");
  const [status, setStatus] = useState("");
  const [branch, setBranch] = useState("");
  const [actor, setActor] = useState("");

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  return (
    <div className="min-h-screen bg-[var(--bg-base)]">
      <header className="h-16 sticky top-0 z-50 bg-white/85 backdrop-blur border-b border-[var(--border)] flex items-center justify-between px-6">
        <div className="flex items-center gap-2 font-extrabold text-lg">
          <span>🐙</span>
          <strong>OpenHub</strong>
        </div>
        <div className="flex items-center gap-4">
          <Link
            href="/07-repo-detail"
            className="px-3 py-1.5 text-sm rounded-md border border-[var(--border)] hover:bg-[var(--bg-muted)]"
          >
            ← リポジトリへ戻る
          </Link>
        </div>
      </header>

      <div className="bg-white border-b border-[#d0d7de] px-6 py-4">
        <div className="text-xl font-semibold">
          <Link href="/07-repo-detail" className="text-[#0969da]">
            openhub
          </Link>
          {" / "}
          <Link href="/07-repo-detail" className="text-[#0969da]">
            <strong>awesome-project</strong>
          </Link>
          <span className="ml-2 inline-block px-2 py-0.5 text-xs rounded-full bg-[var(--info-light)] text-[var(--info)] align-middle">
            Public
          </span>
        </div>
        <nav className="flex gap-1 mt-4">
          <Link href="/07-repo-detail" className="px-4 py-2 text-sm rounded-t-md hover:bg-[#f3f4f6] flex items-center gap-1.5">
            📄 Code
          </Link>
          <Link href="/08-issues-list" className="px-4 py-2 text-sm rounded-t-md hover:bg-[#f3f4f6] flex items-center gap-1.5">
            ⊙ Issues <span className="text-xs text-[var(--text-muted)]">23</span>
          </Link>
          <Link href="/09-pr-list" className="px-4 py-2 text-sm rounded-t-md hover:bg-[#f3f4f6] flex items-center gap-1.5">
            ⇆ Pull requests <span className="text-xs text-[var(--text-muted)]">5</span>
          </Link>
          <Link
            href="/10-actions-list"
            className="px-4 py-2 text-sm rounded-t-md font-semibold border-b-2 border-[#fd8c73] flex items-center gap-1.5"
          >
            ▶ Actions
          </Link>
          <Link href="/07-repo-detail" className="px-4 py-2 text-sm rounded-t-md hover:bg-[#f3f4f6] flex items-center gap-1.5">
            📊 Insights
          </Link>
          <Link href="/07-repo-detail" className="px-4 py-2 text-sm rounded-t-md hover:bg-[#f3f4f6] flex items-center gap-1.5">
            ⚙ Settings
          </Link>
        </nav>
      </div>

      <div className="max-w-[1280px] mx-auto p-6">
        <div className="flex gap-4 mb-4">
          <div className="flex-1 bg-white border border-[#d0d7de] rounded-md px-4 py-3">
            <div className="text-2xl font-semibold text-[#1a7f37]">142</div>
            <div className="text-xs text-[var(--text-secondary)]">Success</div>
          </div>
          <div className="flex-1 bg-white border border-[#d0d7de] rounded-md px-4 py-3">
            <div className="text-2xl font-semibold text-[#cf222e]">8</div>
            <div className="text-xs text-[var(--text-secondary)]">Failed</div>
          </div>
          <div className="flex-1 bg-white border border-[#d0d7de] rounded-md px-4 py-3">
            <div className="text-2xl font-semibold text-[#9a6700]">3</div>
            <div className="text-xs text-[var(--text-secondary)]">In progress</div>
          </div>
          <div className="flex-1 bg-white border border-[#d0d7de] rounded-md px-4 py-3">
            <div className="text-2xl font-semibold text-[var(--text-secondary)]">12</div>
            <div className="text-xs text-[var(--text-secondary)]">Cancelled</div>
          </div>
        </div>

        <div className="grid grid-cols-[256px_1fr] gap-6 mt-6">
          <aside>
            <div className="text-xs uppercase text-[var(--text-secondary)] mb-2">Workflows</div>
            {workflows.map((w) => (
              <Link
                key={w.name}
                href="/10-actions-list"
                className={`block px-3 py-2 rounded-md text-sm mb-0.5 ${
                  w.active
                    ? "bg-[#ddf4ff] text-[#0969da] font-semibold"
                    : "text-[var(--text-primary)] hover:bg-[#f3f4f6]"
                }`}
              >
                {w.name}
              </Link>
            ))}
            <div className="text-xs uppercase text-[var(--text-secondary)] mt-6 mb-2">Management</div>
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
            <form onSubmit={onSubmit} className="flex gap-2 items-center mb-4 flex-wrap">
              <input
                type="text"
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                placeholder="🔍 Filter workflow runs (例: branch:main event:push)"
                className="flex-1 min-w-[240px] px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm"
              />
              <select
                value={event}
                onChange={(e) => setEvent(e.target.value)}
                className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm bg-[#f6f8fa]"
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
                className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm bg-[#f6f8fa]"
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
                className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm bg-[#f6f8fa]"
              >
                <option value="">Branch ▾</option>
                <option>main</option>
                <option>develop</option>
                <option>feature/*</option>
              </select>
              <select
                value={actor}
                onChange={(e) => setActor(e.target.value)}
                className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm bg-[#f6f8fa]"
              >
                <option value="">Actor ▾</option>
                <option>yamada-taro</option>
                <option>suzuki-hanako</option>
              </select>
              <button
                type="button"
                className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm hover:bg-[#f3f4f6]"
              >
                ↕ Sort
              </button>
            </form>

            <div className="bg-white border border-[#d0d7de] rounded-md overflow-hidden">
              <div className="px-4 py-3 bg-[#f6f8fa] border-b border-[#d0d7de] text-sm text-[var(--text-secondary)] flex justify-between items-center">
                <span>
                  <strong className="text-[var(--text-primary)]">168</strong> workflow runs
                </span>
                <span className="text-[var(--text-muted)]">最終更新: 2分前</span>
              </div>

              {runs.map((run, idx) => (
                <div
                  key={run.id}
                  className={`grid grid-cols-[32px_1fr_auto_auto_auto_auto] gap-3 items-center px-4 py-3 border-b border-[#d8dee4] last:border-b-0 ${
                    idx % 2 === 1 ? "bg-[#fafbfc]" : ""
                  }`}
                >
                  <span className={`text-base ${run.iconColor}`}>{run.icon}</span>
                  <div>
                    <div className="text-sm font-semibold">
                      <Link href="/10-actions-list" className="text-[var(--text-primary)] hover:text-[#0969da]">
                        {run.title}
                      </Link>
                    </div>
                    <div className="text-xs text-[var(--text-secondary)] mt-0.5 flex gap-2 items-center flex-wrap">
                      <span>{run.workflow}</span>·
                      <span className="bg-[#ddf4ff] text-[#0969da] px-1.5 py-0.5 rounded font-mono text-[11px]">
                        {run.branch}
                      </span>
                      ·<span>{run.event} by <strong>{run.actor}</strong></span>·
                      <span className="font-mono">{run.sha}</span>
                    </div>
                  </div>
                  <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${run.statusClass}`}>
                    {run.status}
                  </span>
                  <span className="text-xs text-[var(--text-secondary)] min-w-[70px] text-right">{run.duration}</span>
                  <span className="text-xs text-[var(--text-secondary)] min-w-[100px] text-right">{run.time}</span>
                  <div className="flex gap-1">
                    <Link
                      href="/10-actions-list"
                      className="px-2 py-1 border border-[#d0d7de] rounded-md text-xs hover:bg-[#f3f4f6]"
                      title="ログ"
                    >
                      📄
                    </Link>
                    <Link
                      href="/10-actions-list"
                      className="px-2 py-1 border border-[#d0d7de] rounded-md text-xs hover:bg-[#f3f4f6]"
                      title="再実行"
                    >
                      {run.action}
                    </Link>
                  </div>
                </div>
              ))}
            </div>

            <div className="mt-6 rounded-t-md bg-[#161b22] text-[#e6edf3] px-4 py-2 flex justify-between items-center text-[13px]">
              <span>Run #4520 / Test Suite - logs</span>
              <span className="text-[#7d8590]">streaming...</span>
            </div>
            <div className="bg-[#0d1117] text-[#e6edf3] rounded-b-md p-4 font-mono text-xs leading-relaxed max-h-80 overflow-y-auto">
              <div className="whitespace-pre">
                <span className="text-[#7d8590] mr-2">10:42:01</span>
                <span>Run actions/checkout@v4</span>
              </div>
              <div className="whitespace-pre text-[#3fb950]">
                <span className="text-[#7d8590] mr-2">10:42:03</span>
                <span>✓ Checkout complete</span>
              </div>
              <div className="whitespace-pre">
                <span className="text-[#7d8590] mr-2">10:42:05</span>
                <span>Run npm ci</span>
              </div>
              <div className="whitespace-pre text-[#d29922]">
                <span className="text-[#7d8590] mr-2">10:43:11</span>
                <span>⚠ deprecated package detected: lodash.isequal</span>
              </div>
              <div className="whitespace-pre text-[#f85149]">
                <span className="text-[#7d8590] mr-2">10:46:39</span>
                <span>✗ Error: DB connection pool exhausted (test failed)</span>
              </div>
            </div>

            <div className="flex justify-center gap-1 mt-6">
              <Link href="/10-actions-list" className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm hover:bg-[#f3f4f6]">
                ‹ Prev
              </Link>
              <Link href="/10-actions-list" className="px-3 py-1.5 bg-[#0969da] text-white border border-[#0969da] rounded-md text-sm">
                1
              </Link>
              <Link href="/10-actions-list" className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm hover:bg-[#f3f4f6]">
                2
              </Link>
              <Link href="/10-actions-list" className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm hover:bg-[#f3f4f6]">
                3
              </Link>
              <Link href="/10-actions-list" className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm hover:bg-[#f3f4f6]">
                Next ›
              </Link>
            </div>
          </main>
        </div>
      </div>
    </div>
  );
}
