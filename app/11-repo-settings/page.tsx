"use client";

import Link from "next/link";
import { useState } from "react";

export default function RepoSettingsPage() {
  const [repoName, setRepoName] = useState("awesome-project");
  const [description, setDescription] = useState("A curated list of awesome tools and resources");
  const [defaultBranch, setDefaultBranch] = useState("main");
  const [visibility, setVisibility] = useState("public");
  const [issuesOn, setIssuesOn] = useState(true);
  const [wikiOn, setWikiOn] = useState(true);
  const [discussionsOn, setDiscussionsOn] = useState(false);
  const [collabSearch, setCollabSearch] = useState("");

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  const collaborators = [
    { initial: "O", name: "octocat", role: "フルアクセス", access: "Admin", owner: true, gradient: "from-[#fd8c73] to-[#d63384]" },
    { initial: "A", name: "alice-dev", role: "2024年3月から参加", access: "Write", owner: false, gradient: "from-[#0969da] to-[#54aeff]" },
    { initial: "B", name: "bob-coder", role: "2024年5月から参加", access: "Write", owner: false, gradient: "from-[#2da44e] to-[#4ac26b]" },
    { initial: "C", name: "carol-design", role: "2024年8月から参加", access: "Read", owner: false, gradient: "from-[#bf3989] to-[#e85aad]" },
  ];

  const webhooks = [
    { url: "https://ci.example.com/hooks/github", meta: "✓ push, pull_request イベント · 最終配信: 2分前", status: "active" },
    { url: "https://slack.example.com/hooks/abc123", meta: "✓ issues, issue_comment イベント · 最終配信: 1時間前", status: "active" },
    { url: "https://deploy.example.com/webhook", meta: "⚠ release イベント · 最終配信失敗: 502 Bad Gateway", status: "error" },
  ];

  const branches = [
    { name: "main", meta: "PRレビュー必須 (2人) · ステータスチェック必須" },
    { name: "develop", meta: "PRレビュー必須 (1人)" },
  ];

  const navItems = [
    { section: "一般", items: [{ label: "General", active: true }, { label: "Access", active: false }] },
    { section: "コード・自動化", items: [{ label: "Collaborators", active: false }, { label: "Branches", active: false }, { label: "Tags", active: false }, { label: "Actions", active: false }, { label: "Webhooks", active: false }] },
    { section: "セキュリティ", items: [{ label: "Code security", active: false }, { label: "Secrets", active: false }] },
  ];

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      {/* App bar */}
      <header className="sticky top-0 z-50 h-16 bg-white/85 backdrop-blur border-b border-[#d0d7de]">
        <div className="max-w-[1280px] mx-auto px-6 h-full flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link href="/07-repo-detail" className="flex items-center gap-1 font-extrabold text-lg no-underline text-inherit">
              <span>🐙</span>
              <span className="text-gradient bg-gradient-to-r from-[#6366f1] via-[#8b5cf6] to-[#ec4899] bg-clip-text text-transparent">OctoHub</span>
            </Link>
            <span className="text-[#94a3b8]">/</span>
            <Link href="/07-repo-detail" className="text-sm text-[#0969da] no-underline">octocat/awesome-project</Link>
          </div>
          <div className="flex items-center gap-4">
            <Link href="/07-repo-detail" className="text-sm px-3 py-1.5 rounded-md hover:bg-[#f6f8fa] text-[#1f2328] no-underline">← Codeに戻る</Link>
          </div>
        </div>
      </header>

      {/* Repo header */}
      <div className="bg-white border-b border-[#d0d7de] py-4">
        <div className="max-w-[1280px] mx-auto px-6">
          <div className="flex items-center gap-2 text-xl">
            <span>📁</span>
            <Link href="/07-repo-detail" className="text-[#0969da] no-underline">octocat</Link>
            <span className="text-[#94a3b8]">/</span>
            <Link href="/07-repo-detail" className="text-[#0969da] no-underline font-semibold">awesome-project</Link>
            <span className="ml-2 inline-flex items-center text-xs px-2 py-0.5 rounded-full border border-[#d0d7de] bg-[#dbeafe] text-[#1e40af]">Public</span>
          </div>
          <nav className="flex gap-1 mt-4">
            <Link href="/07-repo-detail" className="px-4 py-2 text-sm text-[#1f2328] no-underline rounded-t-md hover:bg-[#f6f8fa]">📄 Code</Link>
            <Link href="/07-repo-detail" className="px-4 py-2 text-sm text-[#1f2328] no-underline rounded-t-md hover:bg-[#f6f8fa]">⊙ Issues <span className="ml-1 text-xs bg-[#f1f5f9] px-1.5 py-0.5 rounded-full">24</span></Link>
            <Link href="/07-repo-detail" className="px-4 py-2 text-sm text-[#1f2328] no-underline rounded-t-md hover:bg-[#f6f8fa]">⇄ Pull requests <span className="ml-1 text-xs bg-[#f1f5f9] px-1.5 py-0.5 rounded-full">7</span></Link>
            <Link href="/07-repo-detail" className="px-4 py-2 text-sm text-[#1f2328] no-underline rounded-t-md hover:bg-[#f6f8fa]">▶ Actions</Link>
            <Link href="/11-repo-settings" className="px-4 py-2 text-sm text-[#1f2328] no-underline rounded-t-md border-b-2 border-[#fd8c73] font-semibold">⚙ Settings</Link>
          </nav>
        </div>
      </div>

      {/* Layout */}
      <div className="max-w-[1280px] mx-auto px-6">
        <div className="grid grid-cols-1 md:grid-cols-[240px_1fr] gap-8 py-6">
          {/* Sidebar */}
          <aside className="bg-white border border-[#d0d7de] rounded-md py-2 h-fit">
            {navItems.map((group) => (
              <div key={group.section}>
                <div className="px-4 py-2 text-xs font-semibold text-[#656d76] uppercase">{group.section}</div>
                {group.items.map((item) => (
                  <Link
                    key={item.label}
                    href="/11-repo-settings"
                    className={`block px-4 py-2 text-sm no-underline border-l-[3px] ${item.active ? "border-[#fd8c73] bg-[#fff8f4] font-semibold text-[#1f2328]" : "border-transparent text-[#1f2328] hover:bg-[#f6f8fa]"}`}
                  >
                    {item.label}
                  </Link>
                ))}
              </div>
            ))}
          </aside>

          {/* Main */}
          <main className="flex flex-col gap-6">
            {/* General */}
            <section className="bg-white border border-[#d0d7de] rounded-md overflow-hidden">
              <div className="px-5 py-4 border-b border-[#d0d7de] bg-[#f6f8fa]">
                <h2 className="text-base font-semibold m-0">一般設定</h2>
                <p className="text-[13px] text-[#656d76] mt-1">リポジトリの基本情報と公開範囲を設定します</p>
              </div>
              <form onSubmit={handleSave}>
                <div className="px-5">
                  <div className="grid grid-cols-1 md:grid-cols-[240px_1fr] gap-4 py-4 border-b border-[#eaeef2] items-center">
                    <div className="text-sm font-semibold">
                      リポジトリ名
                      <div className="text-xs text-[#656d76] font-normal mt-0.5">URLに使用されます</div>
                    </div>
                    <input
                      type="text"
                      value={repoName}
                      onChange={(e) => setRepoName(e.target.value)}
                      className="w-full px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm"
                    />
                  </div>
                  <div className="grid grid-cols-1 md:grid-cols-[240px_1fr] gap-4 py-4 border-b border-[#eaeef2] items-center">
                    <div className="text-sm font-semibold">説明</div>
                    <input
                      type="text"
                      value={description}
                      onChange={(e) => setDescription(e.target.value)}
                      className="w-full px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm"
                    />
                  </div>
                  <div className="grid grid-cols-1 md:grid-cols-[240px_1fr] gap-4 py-4 border-b border-[#eaeef2] items-center">
                    <div className="text-sm font-semibold">デフォルトブランチ</div>
                    <select
                      value={defaultBranch}
                      onChange={(e) => setDefaultBranch(e.target.value)}
                      className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm bg-white"
                    >
                      <option value="main">main</option>
                      <option value="develop">develop</option>
                    </select>
                  </div>
                  <div className="grid grid-cols-1 md:grid-cols-[240px_1fr] gap-4 py-4 border-b border-[#eaeef2] items-center">
                    <div className="text-sm font-semibold">
                      可視性
                      <div className="text-xs text-[#656d76] font-normal mt-0.5">誰がこのリポジトリを見られるか</div>
                    </div>
                    <select
                      value={visibility}
                      onChange={(e) => setVisibility(e.target.value)}
                      className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm bg-white"
                    >
                      <option value="public">Public - 誰でも閲覧可能</option>
                      <option value="private">Private - 招待されたメンバーのみ</option>
                    </select>
                  </div>
                  <ToggleRow label="Issues" on={issuesOn} onToggle={() => setIssuesOn(!issuesOn)} />
                  <ToggleRow label="Wiki" on={wikiOn} onToggle={() => setWikiOn(!wikiOn)} />
                  <ToggleRow label="Discussions" on={discussionsOn} onToggle={() => setDiscussionsOn(!discussionsOn)} last />
                </div>
                <div className="px-5 py-4 bg-[#f6f8fa] border-t border-[#d0d7de] flex justify-end gap-2">
                  <button type="button" className="px-3 py-1.5 text-sm rounded-md border border-[#d0d7de] bg-white hover:bg-[#f6f8fa]">キャンセル</button>
                  <button type="submit" className="px-3 py-1.5 text-sm rounded-md bg-[#2da44e] text-white hover:bg-[#2c974b]">変更を保存</button>
                </div>
              </form>
            </section>

            {/* Collaborators */}
            <section className="bg-white border border-[#d0d7de] rounded-md overflow-hidden">
              <div className="px-5 py-4 border-b border-[#d0d7de] bg-[#f6f8fa]">
                <h2 className="text-base font-semibold m-0">コラボレーター</h2>
                <p className="text-[13px] text-[#656d76] mt-1">このリポジトリに書き込み権限を持つユーザーを管理します</p>
              </div>
              <div className="px-5 py-5">
                <div className="flex gap-2 mb-4">
                  <input
                    type="text"
                    placeholder="ユーザー名またはメールアドレスで検索"
                    value={collabSearch}
                    onChange={(e) => setCollabSearch(e.target.value)}
                    className="flex-1 px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm"
                  />
                  <button className="px-3 py-1.5 text-sm rounded-md bg-[#2da44e] text-white hover:bg-[#2c974b]">+ 追加</button>
                </div>
                <ul>
                  {collaborators.map((c, i) => (
                    <li key={c.name} className={`flex items-center gap-3 py-3 ${i < collaborators.length - 1 ? "border-b border-[#eaeef2]" : ""}`}>
                      <div className={`w-8 h-8 rounded-full bg-gradient-to-br ${c.gradient} text-white flex items-center justify-center font-semibold text-sm`}>{c.initial}</div>
                      <div className="flex-1">
                        <div className="font-semibold text-sm flex items-center gap-2">
                          {c.name}
                          {c.owner && (
                            <span className="text-xs px-2 py-0.5 rounded-full bg-[#eef2ff] text-[#4f46e5]">Owner</span>
                          )}
                        </div>
                        <div className="text-xs text-[#656d76]">{c.role}</div>
                      </div>
                      <select className="w-32 px-2 py-1 border border-[#d0d7de] rounded-md text-sm bg-white" defaultValue={c.access}>
                        {c.owner ? <option>Admin</option> : <>
                          <option>Write</option>
                          <option>Read</option>
                        </>}
                      </select>
                      {!c.owner && (
                        <button className="text-xs px-2 py-1 rounded-md border border-[#cf222e] text-[#cf222e] bg-white hover:bg-[#ffebe9]">削除</button>
                      )}
                    </li>
                  ))}
                </ul>
              </div>
            </section>

            {/* Webhooks */}
            <section className="bg-white border border-[#d0d7de] rounded-md overflow-hidden">
              <div className="px-5 py-4 border-b border-[#d0d7de] bg-[#f6f8fa] flex justify-between items-center">
                <div>
                  <h2 className="text-base font-semibold m-0">Webhooks</h2>
                  <p className="text-[13px] text-[#656d76] mt-1">イベント発生時にHTTP POSTで通知を送信します</p>
                </div>
                <button className="px-3 py-1.5 text-sm rounded-md bg-[#2da44e] text-white hover:bg-[#2c974b]">+ Webhook追加</button>
              </div>
              <div className="px-5 py-5">
                <ul className="flex flex-col gap-2">
                  {webhooks.map((h) => (
                    <li key={h.url} className="px-4 py-3 border border-[#d0d7de] rounded-md flex justify-between items-center">
                      <div>
                        <div className="font-mono text-[13px]">{h.url}</div>
                        <div className="text-xs text-[#656d76] mt-1">{h.meta}</div>
                      </div>
                      <div className="flex gap-2 items-center">
                        {h.status === "active" ? (
                          <span className="text-xs px-2 py-0.5 rounded-full bg-[#d1fae5] text-[#065f46]">Active</span>
                        ) : (
                          <span className="text-xs px-2 py-0.5 rounded-full bg-[#fef3c7] text-[#92400e]">エラー</span>
                        )}
                        <button className="text-xs px-2 py-1 rounded-md border border-[#d0d7de] bg-white hover:bg-[#f6f8fa]">編集</button>
                      </div>
                    </li>
                  ))}
                </ul>
              </div>
            </section>

            {/* Branch protection */}
            <section className="bg-white border border-[#d0d7de] rounded-md overflow-hidden">
              <div className="px-5 py-4 border-b border-[#d0d7de] bg-[#f6f8fa] flex justify-between items-center">
                <div>
                  <h2 className="text-base font-semibold m-0">ブランチ保護ルール</h2>
                  <p className="text-[13px] text-[#656d76] mt-1">特定のブランチへのプッシュやマージに条件を設定します</p>
                </div>
                <button className="px-3 py-1.5 text-sm rounded-md bg-[#2da44e] text-white hover:bg-[#2c974b]">+ ルール追加</button>
              </div>
              <div className="px-5 py-5">
                <ul className="flex flex-col gap-2">
                  {branches.map((b) => (
                    <li key={b.name} className="px-4 py-3 border border-[#d0d7de] rounded-md flex justify-between items-center">
                      <div>
                        <div className="font-mono font-semibold">{b.name}</div>
                        <div className="text-xs text-[#656d76] mt-1">{b.meta}</div>
                      </div>
                      <button className="text-xs px-2 py-1 rounded-md border border-[#d0d7de] bg-white hover:bg-[#f6f8fa]">編集</button>
                    </li>
                  ))}
                </ul>
              </div>
            </section>

            {/* Danger zone */}
            <section className="bg-white border border-[#cf222e] rounded-md overflow-hidden">
              <div className="px-5 py-4 border-b border-[#cf222e] bg-[#ffebe9]">
                <h2 className="text-base font-semibold m-0 text-[#cf222e]">危険な操作</h2>
                <p className="text-[13px] text-[#656d76] mt-1">これらの操作は元に戻せません</p>
              </div>
              <div className="px-5">
                <div className="flex justify-between items-center py-4 border-b border-[#ffebe9]">
                  <div>
                    <div className="font-semibold text-[#cf222e]">可視性の変更</div>
                    <div className="text-[#656d76] text-[13px] mt-0.5">このリポジトリを Private に変更します</div>
                  </div>
                  <button className="text-sm px-3 py-1.5 rounded-md border border-[#cf222e] text-[#cf222e] bg-white hover:bg-[#ffebe9]">可視性を変更</button>
                </div>
                <div className="flex justify-between items-center py-4">
                  <div>
                    <div className="font-semibold text-[#cf222e]">リポジトリを削除</div>
                    <div className="text-[#656d76] text-[13px] mt-0.5">削除後は復元できません</div>
                  </div>
                  <button className="text-sm px-3 py-1.5 rounded-md bg-[#cf222e] text-white hover:bg-[#a40e26]">このリポジトリを削除</button>
                </div>
              </div>
            </section>
          </main>
        </div>
      </div>
    </div>
  );
}

function ToggleRow({ label, on, onToggle, last }: { label: string; on: boolean; onToggle: () => void; last?: boolean }) {
  return (
    <div className={`grid grid-cols-1 md:grid-cols-[240px_1fr] gap-4 py-4 items-center ${last ? "" : "border-b border-[#eaeef2]"}`}>
      <div className="text-sm font-semibold">{label}</div>
      <button
        type="button"
        onClick={onToggle}
        aria-pressed={on}
        className={`relative w-10 h-[22px] rounded-full transition-colors ${on ? "bg-[#2da44e]" : "bg-[#d0d7de]"}`}
      >
        <span className={`absolute top-0.5 w-[18px] h-[18px] bg-white rounded-full transition-all ${on ? "left-[20px]" : "left-0.5"}`} />
      </button>
    </div>
  );
}
