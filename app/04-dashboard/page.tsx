"use client";

import Link from "next/link";
import { useState } from "react";

export default function DashboardPage() {
  const [search, setSearch] = useState("");

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  const metrics = [
    { icon: "📦", label: "リポジトリ", value: "24", delta: "↑ 3 今月", down: false },
    { icon: "⭐", label: "獲得スター", value: "1,248", delta: "↑ 12.5% 先月比", down: false },
    { icon: "🐛", label: "オープンIssue", value: "12", delta: "↑ 4 今週", down: true },
    { icon: "🔀", label: "PR待ち", value: "5", delta: "↓ 2 先週比", down: false },
  ];

  const feedItems = [
    { initials: "SK", gradient: "from-purple-500 to-blue-600", body: (<><Link href="/07-repo-detail" className="text-[color:var(--primary)] font-medium">sato-kenta</Link> が <Link href="/07-repo-detail" className="text-[color:var(--primary)] font-medium">awesome-app</Link> にスターを付けました</>), time: "5分前" },
    { initials: "TM", gradient: "from-red-500 to-pink-600", body: (<><Link href="/13-pr-list" className="text-[color:var(--primary)] font-medium">tanaka-misaki</Link> が <Link href="/07-repo-detail" className="text-[color:var(--primary)] font-medium">react-utils</Link> でPRをオープン: <strong>「useDebounceフックの追加」</strong></>), time: "30分前" },
    { initials: "RH", gradient: "from-green-600 to-green-400", body: (<><Link href="/12-issue-list" className="text-[color:var(--primary)] font-medium">rina-honda</Link> が <Link href="/07-repo-detail" className="text-[color:var(--primary)] font-medium">design-system</Link> でIssueをコメント</>), time: "1時間前" },
    { initials: "YK", gradient: "from-amber-700 to-amber-500", body: (<><Link href="/07-repo-detail" className="text-[color:var(--primary)] font-medium">yuki-kobayashi</Link> が <Link href="/07-repo-detail" className="text-[color:var(--primary)] font-medium">cli-tools</Link> をフォークしました</>), time: "2時間前" },
  ];

  const repos = [
    { name: "yamada-taro/awesome-app", lang: "JavaScript", dot: "bg-yellow-400", stars: 542, forks: 87 },
    { name: "yamada-taro/react-utils", lang: "TypeScript", dot: "bg-blue-600", stars: 328, forks: 42 },
    { name: "yamada-taro/cli-tools", lang: "Go", dot: "bg-cyan-500", stars: 215, forks: 28 },
    { name: "yamada-taro/design-system", lang: "CSS", dot: "bg-purple-700", stars: 163, forks: 19 },
  ];

  const issues = [
    { title: "🟢 ボタンコンポーネントのhover状態が効かない", badge: "bug", badgeClass: "bg-[color:var(--danger-light)] text-[color:var(--danger)]", meta: "design-system #142", time: "2時間前" },
    { title: "🟢 ダークモード対応の要望", badge: "enhancement", badgeClass: "bg-[color:var(--info-light)] text-[color:var(--info)]", meta: "awesome-app #89", time: "昨日" },
    { title: "🟢 ドキュメントの誤字修正", badge: "docs", badgeClass: "bg-[color:var(--success-light)] text-[color:var(--success)]", meta: "react-utils #56", time: "2日前" },
    { title: "🔴 Node 20でのビルドエラー", badge: "bug", badgeClass: "bg-[color:var(--danger-light)] text-[color:var(--danger)]", meta: "cli-tools #34", time: "3日前" },
  ];

  const prs = [
    { title: "🟢 useDebounceフックの追加", badge: "open", badgeClass: "bg-[color:var(--success-light)] text-[color:var(--success)]", meta: "react-utils #67", time: "30分前" },
    { title: "🟣 TypeScript完全対応", badge: "merged", badgeClass: "bg-[color:var(--primary-light)] text-[color:var(--primary)]", meta: "awesome-app #102", time: "6時間前" },
    { title: "🟢 READMEの英訳追加", badge: "open", badgeClass: "bg-[color:var(--success-light)] text-[color:var(--success)]", meta: "design-system #45", time: "昨日" },
    { title: "🟢 パフォーマンス改善: メモ化処理", badge: "review", badgeClass: "bg-[color:var(--warning-light)] text-[color:var(--warning)]", meta: "cli-tools #23", time: "2日前" },
  ];

  return (
    <div className="min-h-screen bg-[color:var(--bg-base)]">
      {/* Topbar */}
      <div className="flex items-center justify-between px-6 py-3 bg-[#24292f] text-white">
        <Link href="/04-dashboard" className="flex items-center gap-2 font-semibold text-white">
          <span>🐙</span> OpenHub
        </Link>
        <form onSubmit={handleSearch} className="flex-1 max-w-md mx-6">
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="🔍 リポジトリ、ユーザー、Issueを検索..."
            className="w-full px-3 py-1.5 rounded-md border border-[#444c56] bg-[#2d333b] text-white placeholder:text-gray-400 focus:outline-none"
          />
        </form>
        <div className="flex items-center gap-3">
          <Link href="/06-repo-create" className="text-white text-sm px-2.5 py-1.5 rounded-md hover:bg-[#2d333b]">＋ 新規</Link>
          <Link href="/12-issue-list" className="text-white text-sm px-2.5 py-1.5 rounded-md hover:bg-[#2d333b]">Issue</Link>
          <Link href="/13-pr-list" className="text-white text-sm px-2.5 py-1.5 rounded-md hover:bg-[#2d333b]">PR</Link>
          <Link href="/15-settings" className="text-white text-sm px-2.5 py-1.5 rounded-md hover:bg-[#2d333b]">⚙️</Link>
          <div className="w-8 h-8 rounded-full bg-gradient-to-br from-purple-500 to-blue-600 flex items-center justify-center text-xs font-semibold">YT</div>
        </div>
      </div>

      <div className="grid grid-cols-[240px_1fr] min-h-[calc(100vh-56px)]">
        {/* Sidebar */}
        <aside className="bg-white border-r border-[color:var(--border)] p-4">
          <div className="pb-4 border-b border-[color:var(--border-subtle)]">
            <div className="text-xs text-[color:var(--text-muted)]">Welcome back</div>
            <div className="font-semibold text-[15px] mt-1">yamada-taro</div>
          </div>

          <div className="mt-4">
            <div className="text-xs font-semibold text-[color:var(--text-muted)] uppercase mb-2">ナビゲーション</div>
            <Link href="/04-dashboard" className="block px-2 py-1.5 rounded-md bg-[color:var(--primary-light)] text-[color:var(--primary)] font-medium text-sm">🏠 ダッシュボード</Link>
            <Link href="/05-repo-list" className="block px-2 py-1.5 rounded-md text-sm hover:bg-[color:var(--bg-muted)] text-[color:var(--text-primary)]">📦 リポジトリ</Link>
            <Link href="/12-issue-list" className="flex items-center justify-between px-2 py-1.5 rounded-md text-sm hover:bg-[color:var(--bg-muted)] text-[color:var(--text-primary)]">
              <span>🐛 Issues</span>
              <span className="text-xs bg-[color:var(--bg-muted)] px-1.5 rounded-full">12</span>
            </Link>
            <Link href="/13-pr-list" className="flex items-center justify-between px-2 py-1.5 rounded-md text-sm hover:bg-[color:var(--bg-muted)] text-[color:var(--text-primary)]">
              <span>🔀 Pull Requests</span>
              <span className="text-xs bg-[color:var(--bg-muted)] px-1.5 rounded-full">5</span>
            </Link>
            <Link href="/15-settings" className="block px-2 py-1.5 rounded-md text-sm hover:bg-[color:var(--bg-muted)] text-[color:var(--text-primary)]">⚙️ 設定</Link>
          </div>

          <div className="mt-6">
            <div className="text-xs font-semibold text-[color:var(--text-muted)] uppercase mb-2">マイリポジトリ</div>
            <Link href="/07-repo-detail" className="block px-2 py-1.5 rounded-md text-sm hover:bg-[color:var(--bg-muted)] text-[color:var(--text-primary)]">📘 awesome-app</Link>
            <Link href="/07-repo-detail" className="block px-2 py-1.5 rounded-md text-sm hover:bg-[color:var(--bg-muted)] text-[color:var(--text-primary)]">📗 react-utils</Link>
            <Link href="/07-repo-detail" className="block px-2 py-1.5 rounded-md text-sm hover:bg-[color:var(--bg-muted)] text-[color:var(--text-primary)]">📙 cli-tools</Link>
            <Link href="/07-repo-detail" className="block px-2 py-1.5 rounded-md text-sm hover:bg-[color:var(--bg-muted)] text-[color:var(--text-primary)]">📕 design-system</Link>
            <Link href="/05-repo-list" className="block px-2 py-1.5 text-[13px] text-[color:var(--primary)]">すべて表示 →</Link>
          </div>

          <div className="mt-6">
            <div className="text-xs font-semibold text-[color:var(--text-muted)] uppercase mb-2">組織</div>
            <Link href="/05-repo-list" className="block px-2 py-1.5 rounded-md text-sm hover:bg-[color:var(--bg-muted)] text-[color:var(--text-primary)]">🏢 acme-corp</Link>
            <Link href="/05-repo-list" className="block px-2 py-1.5 rounded-md text-sm hover:bg-[color:var(--bg-muted)] text-[color:var(--text-primary)]">🏢 open-source-jp</Link>
          </div>
        </aside>

        {/* Main */}
        <main className="p-8">
          <div className="flex justify-between items-center mb-6">
            <div>
              <h1 className="text-2xl font-bold text-[color:var(--text-primary)]">ダッシュボード</h1>
              <div className="text-sm text-[color:var(--text-muted)] mt-1">こんにちは、yamada-taro さん 👋</div>
            </div>
            <Link href="/06-repo-create" className="px-4 py-2 rounded-md bg-[color:var(--primary)] text-white text-sm font-medium hover:bg-[color:var(--primary-hover)]">＋ 新規リポジトリ</Link>
          </div>

          {/* Metrics */}
          <div className="grid grid-cols-4 gap-4 mb-6">
            {metrics.map((m) => (
              <div key={m.label} className="bg-white border border-[color:var(--border)] rounded-xl p-5 shadow-[var(--shadow-xs)]">
                <div className="text-[13px] text-[color:var(--text-secondary)] flex items-center gap-1.5">
                  <span>{m.icon}</span> {m.label}
                </div>
                <div className="text-3xl font-semibold my-2 text-[color:var(--text-primary)]">{m.value}</div>
                <div className={`text-xs ${m.down ? "text-[color:var(--danger)]" : "text-[color:var(--success)]"}`}>{m.delta}</div>
              </div>
            ))}
          </div>

          {/* Content Grid */}
          <div className="grid grid-cols-[60%_40%] gap-5">
            {/* Feed */}
            <div className="bg-white border border-[color:var(--border)] rounded-xl p-5">
              <h3 className="flex justify-between items-center text-base font-semibold mb-3.5 text-[color:var(--text-primary)]">
                📊 アクティビティフィード
                <Link href="/04-dashboard" className="text-[13px] text-[color:var(--primary)] font-normal">すべて表示</Link>
              </h3>
              {feedItems.map((f, i) => (
                <div key={i} className="flex gap-3 py-3 border-b border-[color:var(--border-subtle)] last:border-b-0">
                  <div className={`w-9 h-9 rounded-full bg-gradient-to-br ${f.gradient} text-white flex items-center justify-center font-semibold text-sm flex-shrink-0`}>{f.initials}</div>
                  <div className="text-sm text-[color:var(--text-primary)] flex-1">
                    {f.body}
                    <div className="text-xs text-[color:var(--text-muted)] mt-1">{f.time}</div>
                  </div>
                </div>
              ))}
            </div>

            {/* My Repos */}
            <div className="bg-white border border-[color:var(--border)] rounded-xl p-5">
              <h3 className="flex justify-between items-center text-base font-semibold mb-3.5 text-[color:var(--text-primary)]">
                📦 自分のリポジトリ
                <Link href="/05-repo-list" className="text-[13px] text-[color:var(--primary)] font-normal">すべて</Link>
              </h3>
              {repos.map((r) => (
                <Link key={r.name} href="/07-repo-detail" className="block py-2.5 border-b border-[color:var(--border-subtle)] last:border-b-0">
                  <div className="text-[color:var(--primary)] font-medium text-sm">{r.name}</div>
                  <div className="text-xs text-[color:var(--text-muted)] mt-1 flex gap-3">
                    <span className="flex items-center gap-1"><span className={`inline-block w-2.5 h-2.5 rounded-full ${r.dot}`}></span> {r.lang}</span>
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
              <h3 className="flex justify-between items-center text-base font-semibold mb-3.5 text-[color:var(--text-primary)]">
                🐛 最近のIssue
                <Link href="/12-issue-list" className="text-[13px] text-[color:var(--primary)] font-normal">すべて表示</Link>
              </h3>
              {issues.map((it, i) => (
                <Link key={i} href="/12-issue-list" className="block py-2.5 border-b border-[color:var(--border-subtle)] last:border-b-0">
                  <div className="text-sm text-[color:var(--text-primary)]">{it.title}</div>
                  <div className="text-xs text-[color:var(--text-muted)] mt-1 flex gap-2.5 items-center">
                    <span className={`px-2 py-0.5 rounded-full text-[11px] font-medium ${it.badgeClass}`}>{it.badge}</span>
                    <span>{it.meta}</span>
                    <span>{it.time}</span>
                  </div>
                </Link>
              ))}
            </div>

            <div className="bg-white border border-[color:var(--border)] rounded-xl p-5">
              <h3 className="flex justify-between items-center text-base font-semibold mb-3.5 text-[color:var(--text-primary)]">
                🔀 最近のPull Request
                <Link href="/13-pr-list" className="text-[13px] text-[color:var(--primary)] font-normal">すべて表示</Link>
              </h3>
              {prs.map((pr, i) => (
                <Link key={i} href="/13-pr-list" className="block py-2.5 border-b border-[color:var(--border-subtle)] last:border-b-0">
                  <div className="text-sm text-[color:var(--text-primary)]">{pr.title}</div>
                  <div className="text-xs text-[color:var(--text-muted)] mt-1 flex gap-2.5 items-center">
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
