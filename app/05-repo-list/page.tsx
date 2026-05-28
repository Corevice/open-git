"use client";

import Link from "next/link";
import { useState } from "react";

const repos = [
  {
    id: 1,
    icon: "📘",
    name: "awesome-webapp",
    visibility: "Public",
    desc: "モダンなWebアプリケーションのスターターテンプレート。React + TypeScript + Vite構成。",
    lang: "JavaScript",
    langColor: "#f1e05a",
    stars: "342",
    forks: "28",
    updated: "2時間前",
  },
  {
    id: 2,
    icon: "🔒",
    name: "api-gateway",
    visibility: "Private",
    desc: "マイクロサービス用のAPIゲートウェイ実装。認証・レート制限・ロギング機能を提供。",
    lang: "TypeScript",
    langColor: "#3178c6",
    stars: "87",
    forks: "12",
    updated: "5時間前",
  },
  {
    id: 3,
    icon: "📘",
    name: "data-pipeline",
    visibility: "Public",
    desc: "大規模データ処理パイプライン。Apache Sparkベースの分散処理フレームワーク。",
    lang: "Python",
    langColor: "#3572A5",
    stars: "1.2k",
    forks: "156",
    updated: "1日前",
  },
  {
    id: 4,
    icon: "🔒",
    name: "internal-tools",
    visibility: "Internal",
    desc: "社内向け開発ツール集。CLIユーティリティとスクリプトのコレクション。",
    lang: "Go",
    langColor: "#00ADD8",
    stars: "24",
    forks: "6",
    updated: "2日前",
  },
  {
    id: 5,
    icon: "📘",
    name: "rust-cli-tools",
    visibility: "Public",
    desc: "高速なCLIツールセット。ファイル操作・テキスト処理・ネットワーク診断機能を提供。",
    lang: "Rust",
    langColor: "#dea584",
    stars: "567",
    forks: "43",
    updated: "3日前",
  },
];

function visibilityBadge(v: string) {
  if (v === "Public") return "bg-[var(--success-light)] text-[var(--success)]";
  if (v === "Private") return "bg-[var(--warning-light)] text-[var(--warning)]";
  return "bg-[var(--info-light)] text-[var(--info)]";
}

export default function Page() {
  const [search, setSearch] = useState("");
  const [lang, setLang] = useState("すべての言語");
  const [visibility, setVisibility] = useState("すべての可視性");
  const [sort, setSort] = useState("更新日順");

  return (
    <div className="min-h-screen bg-[var(--bg-base)]">
      <header className="h-16 sticky top-0 z-50 flex items-center justify-between px-6 bg-white/85 backdrop-blur border-b border-[var(--border)]">
        <div className="flex items-center gap-2 text-lg font-extrabold">
          <span>🐙</span>
          <span>OpenHub</span>
        </div>
        <div className="flex items-center gap-2">
          <Link href="/08-search-global" className="px-3 py-1.5 text-sm rounded-md hover:bg-[var(--bg-muted)] text-[var(--text-secondary)]">🔍 検索</Link>
          <Link href="/06-repo-create" className="px-3 py-1.5 text-sm rounded-md hover:bg-[var(--bg-muted)] text-[var(--text-secondary)]">＋</Link>
          <Link href="/04-dashboard" className="px-3 py-1.5 text-sm rounded-md hover:bg-[var(--bg-muted)] text-[var(--text-secondary)]">🏠 ダッシュボード</Link>
          <span className="px-2.5 py-1 text-xs rounded-full bg-[var(--primary-light)] text-[var(--primary)] font-semibold">@taro</span>
        </div>
      </header>

      <div className="flex min-h-[calc(100vh-64px)]">
        <aside className="w-60 bg-white border-r border-[var(--border)] py-5">
          <div className="px-4 mb-6">
            <div className="text-xs uppercase tracking-wider text-[var(--text-muted)] font-semibold mb-2">ナビゲーション</div>
            <Link href="/04-dashboard" className="flex items-center gap-2 px-3 py-2 rounded-md text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-muted)]">📊 ダッシュボード</Link>
            <Link href="/05-repo-list" className="flex items-center justify-between gap-2 px-3 py-2 rounded-md text-sm bg-[var(--primary-light)] text-[var(--primary)] font-medium">
              <span>📁 リポジトリ</span>
              <span className="text-xs bg-white px-1.5 rounded">24</span>
            </Link>
            <Link href="/08-search-global" className="flex items-center gap-2 px-3 py-2 rounded-md text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-muted)]">🔍 検索</Link>
            <Link href="/14-import-wizard" className="flex items-center gap-2 px-3 py-2 rounded-md text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-muted)]">📥 インポート</Link>
          </div>
          <div className="px-4">
            <div className="text-xs uppercase tracking-wider text-[var(--text-muted)] font-semibold mb-2">組織</div>
            <Link href="/05-repo-list" className="flex items-center gap-2 px-3 py-2 rounded-md text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-muted)]">🏢 acme-corp</Link>
            <Link href="/05-repo-list" className="flex items-center gap-2 px-3 py-2 rounded-md text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-muted)]">🏢 openhub-org</Link>
          </div>
        </aside>

        <main className="flex-1 p-8 max-w-[1280px]">
          <div className="flex justify-between items-center mb-6">
            <div>
              <h1 className="text-2xl font-semibold m-0">リポジトリ</h1>
              <p className="text-[var(--text-muted)] text-sm mt-1">24個のリポジトリがあります</p>
            </div>
            <div className="flex gap-2">
              <Link href="/14-import-wizard" className="px-3 py-2 rounded-md text-sm border border-[var(--border)] bg-white hover:bg-[var(--bg-muted)]">📥 インポート</Link>
              <Link href="/06-repo-create" className="px-3 py-2 rounded-md text-sm bg-[var(--primary)] text-white hover:bg-[var(--primary-hover)]">＋ 新規作成</Link>
            </div>
          </div>

          <form
            onSubmit={(e) => {
              e.preventDefault();
              // TODO: wire to API
            }}
            className="flex gap-3 items-center mb-5 p-3 bg-white border border-[var(--border)] rounded-lg"
          >
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="🔍 リポジトリを検索..."
              className="flex-1 px-3 py-2 border border-[var(--border)] rounded-md text-sm"
            />
            <select value={lang} onChange={(e) => setLang(e.target.value)} className="px-3 py-2 border border-[var(--border)] rounded-md bg-[var(--bg-muted)] text-sm min-w-[140px]">
              <option>すべての言語</option>
              <option>JavaScript</option>
              <option>TypeScript</option>
              <option>Python</option>
              <option>Go</option>
              <option>Rust</option>
            </select>
            <select value={visibility} onChange={(e) => setVisibility(e.target.value)} className="px-3 py-2 border border-[var(--border)] rounded-md bg-[var(--bg-muted)] text-sm min-w-[140px]">
              <option>すべての可視性</option>
              <option>Public</option>
              <option>Private</option>
              <option>Internal</option>
            </select>
            <select value={sort} onChange={(e) => setSort(e.target.value)} className="px-3 py-2 border border-[var(--border)] rounded-md bg-[var(--bg-muted)] text-sm min-w-[140px]">
              <option>更新日順</option>
              <option>作成日順</option>
              <option>名前順</option>
              <option>スター数順</option>
            </select>
          </form>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {repos.map((r) => (
              <Link
                key={r.id}
                href="/07-repo-detail"
                className="block bg-white border border-[var(--border)] rounded-lg p-4 hover:border-[var(--primary)] transition-colors"
              >
                <div className="flex justify-between items-start mb-2">
                  <h3 className="text-base font-semibold text-[var(--primary)] m-0">
                    {r.icon} {r.name}
                  </h3>
                  <span className={`px-2 py-0.5 text-xs rounded-full font-semibold ${visibilityBadge(r.visibility)}`}>
                    {r.visibility}
                  </span>
                </div>
                <p className="text-[13px] text-[var(--text-secondary)] my-2 leading-relaxed">{r.desc}</p>
                <div className="flex gap-4 text-xs text-[var(--text-muted)] items-center">
                  <span className="flex items-center gap-1">
                    <span className="inline-block w-2.5 h-2.5 rounded-full" style={{}} >
                      <span className="inline-block w-2.5 h-2.5 rounded-full" />
                    </span>
                    <span className={`inline-block w-2.5 h-2.5 rounded-full [background:${r.langColor}]`} />
                    {r.lang}
                  </span>
                  <span>⭐ {r.stars}</span>
                  <span>🍴 {r.forks}</span>
                  <span>更新: {r.updated}</span>
                </div>
              </Link>
            ))}
          </div>

          <div className="flex justify-center gap-1 mt-8">
            <Link href="/05-repo-list" className="px-3 py-1.5 border border-[var(--border)] bg-white rounded-md text-sm text-[var(--text-primary)] hover:bg-[var(--bg-muted)]">‹ 前へ</Link>
            <Link href="/05-repo-list" className="px-3 py-1.5 border border-[var(--primary)] bg-[var(--primary)] text-white rounded-md text-sm">1</Link>
            <Link href="/05-repo-list" className="px-3 py-1.5 border border-[var(--border)] bg-white rounded-md text-sm text-[var(--text-primary)] hover:bg-[var(--bg-muted)]">2</Link>
            <Link href="/05-repo-list" className="px-3 py-1.5 border border-[var(--border)] bg-white rounded-md text-sm text-[var(--text-primary)] hover:bg-[var(--bg-muted)]">3</Link>
            <Link href="/05-repo-list" className="px-3 py-1.5 border border-[var(--border)] bg-white rounded-md text-sm text-[var(--text-primary)] hover:bg-[var(--bg-muted)]">次へ ›</Link>
          </div>
        </main>
      </div>
    </div>
  );
}
