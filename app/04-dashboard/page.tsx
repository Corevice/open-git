"use client";

import Link from "next/link";
import { useState } from "react";

const metrics = [
  { label: "📦 リポジトリ", value: "24", delta: "↑ 3 今月", down: false },
  { label: "⭐ 獲得スター", value: "1,248", delta: "↑ 12.5% 先月比", down: false },
  { label: "🐛 オープンIssue", value: "12", delta: "↑ 4 今週", down: true },
  { label: "🔀 PR待ち", value: "5", delta: "↓ 2 先週比", down: false },
];

const feed = [
  { initials: "SK", gradient: "from-purple-600 to-blue-600", text: "がリポジトリにスターを付けました", user: "sato-kenta", target: "awesome-app", time: "5分前" },
  { initials: "TM", gradient: "from-red-600 to-pink-600", text: "がPRをオープンしました: 「useDebounceフックの追加」", user: "tanaka-misaki", target: "react-utils", time: "30分前" },
  { initials: "RH", gradient: "from-green-700 to-green-500", text: "がIssueをコメント: 「ボタンのhover状態について」", user: "rina-honda", target: "design-system", time: "1時間前" },
  { initials: "YK", gradient: "from-yellow-700 to-yellow-500", text: "がリポジトリをフォークしました", user: "yuki-kobayashi", target: "cli-tools", time: "2時間前" },
  { initials: "AM", gradient: "from-blue-600 to-blue-400", text: "が main ブランチに 3 コミットをプッシュ", user: "aoki-masaru", target: "", time: "4時間前" },
];

const myRepos = [
  { name: "yamada-taro/awesome-app", lang: "JavaScript", color: "bg-yellow-400", stars: 542, forks: 87 },
  { name: "yamada-taro/react-utils", lang: "TypeScript", color: "bg-blue-600", stars: 328, forks: 42 },
  { name: "yamada-taro/cli-tools", lang: "Go", color: "bg-cyan-500", stars: 215, forks: 28 },
  { name: "yamada-taro/design-system", lang: "CSS", color: "bg-purple-800", stars: 163, forks: 19 },
];

const recentIssues = [
  { title: "🟢 ボタンコンポーネントのhover状態が効かない", badge: "bug", badgeClass: "bg-red-100 text-red-700", meta: "design-system #142", time: "2時間前" },
  { title: "🟢 ダークモード対応の要望", badge: "enhancement", badgeClass: "bg-blue-100 text-blue-700", meta: "awesome-app #89", time: "昨日" },
  { title: "🟢 ドキュメントの誤字修正", badge: "docs", badgeClass: "bg-green-100 text-green-700", meta: "react-utils #56", time: "2日前" },
  { title: "🔴 Node 20でのビルドエラー", badge: "bug", badgeClass: "bg-red-100 text-red-700", meta: "cli-tools #34", time: "3日前" },
];

const recentPRs = [
  { title: "🟢 useDebounceフックの追加", badge: "open", badgeClass: "bg-green-100 text-green-700", meta: "react-utils #67", time: "30分前" },
  { title: "🟣 TypeScript完全対応", badge: "merged", badgeClass: "bg-purple-100 text-purple-700", meta: "awesome-app #102", time: "6時間前" },
  { title: "🟢 READMEの英訳追加", badge: "open", badgeClass: "bg-green-100 text-green-700", meta: "design-system #45", time: "昨日" },
  { title: "🟢 パフォーマンス改善: メモ化処理", badge: "review", badgeClass: "bg-yellow-100 text-yellow-700", meta: "cli-tools #23", time: "2日前" },
];

export default function DashboardPage() {
  const [search, setSearch] = useState("");

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  return (
    <div className="min-h-screen bg-[color:var(--bg-base)]">
      {/* Topbar */}
      <div className="flex items-center justify-between px-6 py-3 bg-[#24292f] text-white">
        <div className="flex items-center gap-2 font-semibold">
          <span>🐙</span> OpenHub
        </div>
        <form onSubmit={handleSearch} className="flex-1 max-w-md mx-6">
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="🔍 リポジトリ、ユーザー、Issueを検索..."
            className="w-full px-3 py-1.5 rounded-md border border-[#444c56] bg-[#2d333b] text-white text-sm"
          />
        </form>
        <div className="flex items-center gap-3 text-sm">
          <Link href="/06-repo-create" className="text-white hover:bg-[#2d333b] px-2.5 py-1.5 rounded-md">＋ 新規</Link>
          <Link href="/12-issue-list" className="text-white hover:bg-[#2d333b] px-2.5 py-1.5 rounded-md">Issue</Link>
          <Link href="/13-pr-list" className="text-white hover:bg-[#2d333b] px-2.5 py-1.5 rounded-md">PR</Link>
          <Link href="/15-settings" className="text-white hover:bg-[#2d333b] px-2.5 py-1.5 rounded-md">⚙️</Link>
          <div className="w-8 h-8 rounded-full bg-gradient-to-br from-purple-600 to-blue-600 flex items-center justify-center font-semibold text-xs">YT</div>
        </div>
      </div>

      <div className="grid grid-cols-[240px_1fr] min-h-screen">
        {/* Sidebar */}
        <aside className="bg-white border-r border-[color:var(--border)] p-4">
          <div className="pb-4 border-b border-[color:var(--border)]">
            <div className="text-xs text-[color:var(--text-muted)]">Welcome back</div>
            <div className="font-semibold text-[15px] mt-1">yamada-taro</div>
          </div>

          <div className="mt-4">
            <div className="text-xs font-semibold text-[color:var(--text-muted)] uppercase mb-2">ナビゲーション</div>
            <Link href="/04-dashboard" className="block px-2 py-1.5 rounded-md bg-[color:var(--primary-light)] text-[color:var(--primary)] text-sm font-medium">🏠 ダッシュボード</Link>
            <Link href="/05-repo-list" className="block px-2 py-1.5 rounded-md hover:bg-[color:var(--bg-muted)] text-sm">📦 リポジトリ</Link>
            <Link href="/12-issue-list" className="flex justify-between px-2 py-1.5 rounded-md hover:bg-[color:var(--bg-muted)] text-sm">
              <span>🐛 Issues</span>
              <span className="text-xs bg-[color:var(--bg-muted)] px-1.5 rounded-full">12</span>
            </Link>
            <Link href="/13-pr-list" className="flex justify-between px-2 py-1.5 rounded-md hover:bg-[color:var(--bg-muted)] text-sm">
              <span>🔀 Pull Requests</span>
              <span className="text-xs bg-[color:var(--bg-muted)] px-1.5 rounded-full">5</span>
            </Link>
            <Link href="/15-settings" className="block px-2 py-1.5 rounded-md hover:bg-[color:var(--bg-muted)] text-sm">⚙️ 設定</Link>
          </div>

          <div className="mt-4">
            <div className="text-xs font-semibold text-[color:var(--text-muted)] uppercase mb-2">マイリポジトリ</div>
            <Link href="/07-repo-detail" className="block px-2 py-1.5 rounded-md hover:bg-[color:var(--bg-muted)] text-sm">📘 awesome-app</Link>
            <Link href="/07-repo-detail" className="block px-2 py-1.5 rounded-md hover:bg-[color:var(--bg-muted)] text-sm">📗 react-utils</Link>
            <Link href="/07-repo-detail" className="block px-2 py-1.5 rounded-md hover:bg-[color:var(--bg-muted)] text-sm">📙 cli-tools</Link>
            <Link href="/05-repo-list" className="block px-2 py-1.5 rounded-md text-[color:var(--primary)] text-[13px]">すべて表示 →</Link>
          </div>

          <div className="mt-4">
            <div className="text-xs font-semibold text-[color:var(--text-muted)] uppercase mb-2">組織</div>
            <Link href="/05-repo-list" className="block px-2 py-1.5 rounded-md hover:bg-[color:var(--bg-muted)] text-sm">🏢 acme-corp</Link>
            <Link href="/05-repo-list" className="block px-2 py-1.5 rounded-md hover:bg-[color:var(--bg-muted)] text-sm">🏢 open-source-jp</Link>
          </div>
        </aside>

        {/* Main */}
        <main className="p-8">
          <div className="flex justify-between items-center mb-6">
            <div>
              <h1 className="text-2xl font-bold text-[color:var(--text-primary)]">ダッシュボード</h1>
              <div className="text-sm text-[color:var(--text-muted)] mt-1">こんにちは、yamada-taro さん 👋</div>
            </div>
            <Link
              href="/06-repo-create"
              className="px-4 py-2 bg-[color:var(--primary)] hover:bg-[color:var(--primary-hover)] text-white rounded-md text-sm font-medium"
            >
              ＋ 新規リポジトリ
            </Link>
          </div>

          {/* Metrics */}
          <div className="grid grid-cols-4 gap-4 mb-6">
            {metrics.map((m) => (
              <div key={m.label} className="bg-white border border-[color:var(--border)] rounded-xl p-5 shadow-[var(--shadow-xs)]">
                <div className="text-[13px] text-[color:var(--text-secondary)] flex items-center gap-1.5">{m.label}</div>
                <div className="text-3xl font-semibold my-2 text-[color:var(--text-primary)]">{m.value}</div>
                <div className={`text-xs ${m.down ? "text-[color:var(--danger)]" : "text-[color:var(--success)]"}`}>{m.delta}</div>
              </div>
            ))}
          </div>

          {/* Content Grid */}
          <div className="grid grid-cols-[60%_40%] gap-5">
            {/* Activity Feed */}
            <div className="bg-white border border-[color:var(--border)] rounded-xl p-5">
              <h3 className="text-base font-semibold mb-3 flex justify-between items-center">
                📊 アクティビティフィード
                <Link href="#" className="text-[13px] text-[color:var(--primary)] font-normal">すべて表示</Link>
              </h3>
              {feed.map((f, i) => (
                <div key={i} className="flex gap-3 py-3 border-b border-[color:var(--border-subtle)] last:border-none">
                  <div className={`w-9 h-9 rounded-full bg-gradient-to-br ${f.gradient} text-white flex items-center justify-center font-semibold flex-shrink-0 text-sm`}>
                    {f.initials}
                  </div>
                  <div className="text-sm flex-1">
                    <Link href="/07-repo-detail" className="text-[color:var(--primary)] font-medium">{f.user}</Link>
                    {f.target && (
                      <>
                        {" が "}
                        <Link href="/07-repo-detail" className="text-[color:var(--primary)] font-medium">{f.target}</Link>
                      </>
                    )}
                    {" "}{f.text}
                    <div className="text-xs text-[color:var(--text-muted)] mt-1">{f.time}</div>
                  </div>
                </div>
              ))}
            </div>

            {/* My Repositories */}
            <div className="bg-white border border-[color:var(--border)] rounded-xl p-5">
              <h3 className="text-base font-semibold mb-3 flex justify-between items-center">
                📦 自分のリポジトリ
                <Link href="/05-repo-list" className="text-[13px] text-[color:var(--primary)] font-normal">すべて</Link>
              </h3>
              {myRepos.map((r) => (
                <Link key={r.name} href="/07-repo-detail" className="block py-2.5 border-b border-[color:var(--border-subtle)] last:border-none">
                  <div className="text-[color:var(--primary)] font-medium text-sm">{r.name}</div>
                  <div className="text-xs text-[color:var(--text-muted)] mt-1 flex gap-3">
                    <span className="flex items-center gap-1.5">
                      <span className={`inline-block w-2.5 h-2.5 rounded-full ${r.color}`} /> {r.lang}
                    </span>
                    <span>⭐ {r.stars}</span>
                    <span>🍴 {r.forks}</span>
                  </div>
                </Link>
              ))}
            </div>
          </div>

          {/* Row 2 */}
          <div className="grid grid-cols-2 gap-5 mt-5">
            <div className="bg-white border border-[color:var(--border)] rounded-xl p-5">
              <h3 className="text-base font-semibold mb-3 flex justify-between items-center">
                🐛 最近のIssue
                <Link href="/12-issue-list" className="text-[13px] text-[color:var(--primary)] font-normal">すべて表示</Link>
              </h3>
              {recentIssues.map((it, i) => (
                <Link key={i} href="/12-issue-list" className="block py-2.5 border-b border-[color:var(--border-subtle)] last:border-none">
                  <div className="text-sm text-[color:var(--text-primary)]">{it.title}</div>
                  <div className="text-xs text-[color:var(--text-muted)] mt-1 flex gap-2 items-center">
                    <span className={`px-2 py-0.5 rounded-full text-[11px] font-medium ${it.badgeClass}`}>{it.badge}</span>
                    <span>{it.meta}</span>
                    <span>{it.time}</span>
                  </div>
                </Link>
              ))}
            </div>

            <div className="bg-white border border-[color:var(--border)] rounded-xl p-5">
              <h3 className="text-base font-semibold mb-3 flex justify-between items-center">
                🔀 最近のPull Request
                <Link href="/13-pr-list" className="text-[13px] text-[color:var(--primary)] font-normal">すべて表示</Link>
              </h3>
              {recentPRs.map((pr, i) => (
                <Link key={i} href="/13-pr-list" className="block py-2.5 border-b border-[color:var(--border-subtle)] last:border-none">
                  <div className="text-sm text-[color:var(--text-primary)]">{pr.title}</div>
                  <div className="text-xs text-[color:var(--text-muted)] mt-1 flex gap-2 items-center">
                    <span className={`px-2 py-0.5 rounded-full text-[11px] font-medium ${pr.badgeClass}`}>{pr.badge}</span>
                    <span>{pr.meta}</span>
                    <span>{pr.time}</span>
                  </div>
                </Link>
              ))}
            </div>
          </div>
        </main>
      </div>
    </div>
  );
}
