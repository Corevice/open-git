"use client";

import Link from "next/link";
import { useState } from "react";

type Collaborator = {
  initial: string;
  name: string;
  role: string;
  permission: string;
  isOwner?: boolean;
  avatarClass: string;
};

type Webhook = {
  url: string;
  meta: string;
  status: "Active" | "エラー";
};

const collaborators: Collaborator[] = [
  { initial: "O", name: "octocat", role: "フルアクセス", permission: "Admin", isOwner: true, avatarClass: "bg-gradient-to-br from-[#fd8c73] to-[#d63384]" },
  { initial: "A", name: "alice-dev", role: "2024年3月から参加", permission: "Write", avatarClass: "bg-gradient-to-br from-[#0969da] to-[#54aeff]" },
  { initial: "B", name: "bob-coder", role: "2024年5月から参加", permission: "Write", avatarClass: "bg-gradient-to-br from-[#2da44e] to-[#4ac26b]" },
  { initial: "C", name: "carol-design", role: "2024年8月から参加", permission: "Read", avatarClass: "bg-gradient-to-br from-[#bf3989] to-[#e85aad]" },
];

const webhooks: Webhook[] = [
  { url: "https://ci.example.com/hooks/github", meta: "✓ push, pull_request イベント · 最終配信: 2分前", status: "Active" },
  { url: "https://slack.example.com/hooks/abc123", meta: "✓ issues, issue_comment イベント · 最終配信: 1時間前", status: "Active" },
  { url: "https://deploy.example.com/webhook", meta: "⚠ release イベント · 最終配信失敗: 502 Bad Gateway", status: "エラー" },
];

export default function RepoSettingsPage() {
  const [repoName, setRepoName] = useState("awesome-project");
  const [description, setDescription] = useState("A curated list of awesome tools and resources");
  const [defaultBranch, setDefaultBranch] = useState("main");
  const [visibility, setVisibility] = useState("Public");
  const [issuesOn, setIssuesOn] = useState(true);
  const [wikiOn, setWikiOn] = useState(true);
  const [discussionsOn, setDiscussionsOn] = useState(false);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      {/* App Bar */}
      <header className="sticky top-0 z-50 h-16 bg-white/85 backdrop-blur border-b border-[#d0d7de]">
        <div className="max-w-[1280px] mx-auto px-6 h-full flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link href="/07-repo-detail" className="flex items-center gap-2 font-extrabold text-lg text-inherit no-underline">
              <span>🐙</span>
              <span className="bg-gradient-to-r from-[#6366f1] via-[#8b5cf6] to-[#ec4899] bg-clip-text text-transparent">OctoHub</span>
            </Link>
            <span className="text-[#94a3b8]">/</span>
            <Link href="/07-repo-detail" className="text-sm text-[#0969da] no-underline">octocat/awesome-project</Link>
          </div>
          <div className="flex items-center gap-3">
            <Link href="/07-repo-detail" className="text-sm px-3 py-1.5 rounded-md hover:bg-[#f6f8fa] text-[#1f2328]">← Codeに戻る</Link>
          </div>
        </div>
      </header>

      {/* Repo Header */}
      <div className="bg-white border-b border-[#d0d7de] py-4">
        <div className="max-w-[1280px] mx-auto px-6">
          <div className="flex items-center gap-2 text-xl">
            <span>📁</span>
            <Link href="/07-repo-detail" className="text-[#0969da] no-underline">octocat</Link>
            <span className="text-[#94a3b8]">/</span>
            <Link href="/07-repo-detail" className="text-[#0969da] font-semibold no-underline">awesome-project</Link>
            <span className="ml-2 text-xs px-2 py-0.5 rounded-full bg-[#ddf4ff] text-[#0969da] border border-[#54aeff]">Public</span>
          </div>
          <nav className="flex gap-1 mt-4">
            <Link href="/07-repo-detail" className="px-4 py-2 text-sm text-[#1f2328] rounded-t-md">📄 Code</Link>
            <Link href="/07-repo-detail" className="px-4 py-2 text-sm text-[#1f2328] rounded-t-md">⊙ Issues <span className="ml-1 text-xs px-1.5 py-0.5 rounded-full bg-[#eaeef2]">24</span></Link>
            <Link href="/07-repo-detail" className="px-4 py-2 text-sm text-[#1f2328] rounded-t-md">⇄ Pull requests <span className="ml-1 text-xs px-1.5 py-0.5 rounded-full bg-[#eaeef2]">7</span></Link>
            <Link href="/07-repo-detail" className="px-4 py-2 text-sm text-[#1f2328] rounded-t-md">▶ Actions</Link>
            <Link href="/11-repo-settings" className="px-4 py-2 text-sm rounded-t-md border-b-2 border-[#fd8c73] font-semibold">⚙ Settings</Link>
          </nav>
        </div>
      </div>

      <div className="max-w-[1280px] mx-auto px-6">
        <div className="grid grid-cols-[240px_1fr] gap-8 py-6">
          {/* Settings Nav */}
          <aside className="bg-white border border-[#d0d7de] rounded-md py-2 h-fit">
            <div className="px-4 py-2 text-xs font-semibold text-[#656d76] uppercase">一般</div>
            <Link href="/11-repo-settings" className="block px-4 py-2 text-sm text-[#1f2328] border-l-[3px] border-[#fd8c73] bg-[#fff8f4] font-semibold no-underline">General</Link>
            <Link href="/11-repo-settings" className="block px-4 py-2 text-sm text-[#1f2328] border-l-[3px] border-transparent hover:bg-[#f6f8fa] no-underline">Access</Link>
            <div className="px-4 py-2 text-xs font-semibold text-[#656d76] uppercase">コード・自動化</div>
            <Link href="/11-repo-settings" className="block px-4 py-2 text-sm text-[#1f2328] border-l-[3px] border-transparent hover:bg-[#f6f8fa] no-underline">Collaborators</Link>
            <Link href="/11-repo-settings" className="block px-4 py-2 text-sm text-[#1f2328] border-l-[3px] border-transparent hover:bg-[#f6f8fa] no-underline">Branches</Link>
            <Link href="/11-repo-settings" className="block px-4 py-2 text-sm text-[#1f2328] border-l-[3px] border-transparent hover:bg-[#f6f8fa] no-underline">Tags</Link>
            <Link href="/11-repo-settings" className="block px-4 py-2 text-sm text-[#1f2328] border-l-[3px] border-transparent hover:bg-[#f6f8fa] no-underline">Actions</Link>
            <Link href="/11-repo-settings" className="block px-4 py-2 text-sm text-[#1f2328] border-l-[3px] border-transparent hover:bg-[#f6f8fa] no-underline">Webhooks</Link>
            <div className="px-4 py-2 text-xs font-semibold text-[#656d76] uppercase">セキュリティ</div>
            <Link href="/11-repo-settings" className="block px-4 py-2 text-sm text-[#1f2328] border-l-[3px] border-transparent hover:bg-[#f6f8fa] no-underline">Code security</Link>
            <Link href="/11-repo-settings" className="block px-4 py-2 text-sm text-[#1f2328] border-l-[3px] border-transparent hover:bg-[#f6f8fa] no-underline">Secrets</Link>
          </aside>

          {/* Main */}
          <main className="flex flex-col gap-6">
            {/* General */}
            <section className="bg-white border border-[#d0d7de] rounded-md overflow-hidden">
              <div className="px-5 py-4 border-b border-[#d0d7de] bg-[#f6f8fa]">
                <h2 className="text-base font-semibold m-0">一般設定</h2>
                <p className="text-[13px] text-[#656d76] mt-1">リポジトリの基本情報と公開範囲を設定します</p>
              </div>
              <form onSubmit={handleSubmit}>
                <div className="px-5">
                  <div className="grid grid-cols-[240px_1fr] gap-4 py-4 border-b border-[#eaeef2] items-center">
                    <div className="text-sm font-semibold">
                      リポジトリ名
                      <div className="text-xs text-[#656d76] font-normal mt-0.5">URLに使用されます</div>
                    </div>
                    <input type="text" className="w-full px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm" value={repoName} onChange={(e) => setRepoName(e.target.value)} />
                  </div>
                  <div className="grid grid-cols-[240px_1fr] gap-4 py-4 border-b border-[#eaeef2] items-center">
                    <div className="text-sm font-semibold">説明</div>
                    <input type="text" className="w-full px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm" value={description} onChange={(e) => setDescription(e.target.value)} />
                  </div>
                  <div className="grid grid-cols-[240px_1fr] gap-4 py-4 border-b border-[#eaeef2] items-center">
                    <div className="text-sm font-semibold">デフォルトブランチ</div>
                    <select className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm bg-white" value={defaultBranch} onChange={(e) => setDefaultBranch(e.target.value)}>
                      <option value="main">main</option>
                      <option value="develop">develop</option>
                    </select>
                  </div>
                  <div className="grid grid-cols-[240px_1fr] gap-4 py-4 border-b border-[#eaeef2] items-center">
                    <div className="text-sm font-semibold">
                      可視性
                      <div className="text-xs text-[#656d76] font-normal mt-0.5">誰がこのリポジトリを見られるか</div>
                    </div>
                    <select className="px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm bg-white" value={visibility} onChange={(e) => setVisibility(e.target.value)}>
                      <option value="Public">Public - 誰でも閲覧可能</option>
                      <option value="Private">Private - 招待されたメンバーのみ</option>
                    </select>
                  </div>
                  <Toggle label="Issues" on={issuesOn} onToggle={() => setIssuesOn((v) => !v)} />
                  <Toggle label="Wiki" on={wikiOn} onToggle={() => setWikiOn((v) => !v)} />
                  <Toggle label="Discussions" on={discussionsOn} onToggle={() => setDiscussionsOn((v) => !v)} last />
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
              <div className="p-5">
                <div className="flex gap-2 mb-4">
                  <input type="text" className="flex-1 px-3 py-1.5 border border-[#d0d7de] rounded-md text-sm" placeholder="ユーザー名またはメールアドレスで検索" />
                  <button className="px-3 py-1.5 text-sm rounded-md bg-[#2da44e] text-white hover:bg-[#2c974b]">+ 追加</button>
                </div>
                <ul className="m-0 p-0">
                  {collaborators.map((c, i) => (
                    <li key={c.name} className={`flex items-center gap-3 py-3 ${i < collaborators.length - 1 ? "border-b border-[#eaeef2]" : ""}`}>
                      <div className={`w-8 h-8 rounded-full flex items-center justify-center text-white font-semibold text-sm ${c.avatarClass}`}>{c.initial}</div>
                      <div className="flex-1">
                        <div className="font-semibold text-sm flex items-center gap-2">
                          {c.name}
                          {c.isOwner && <span className="text-xs px-2 py-0.5 rounded-full bg-[#ddf4ff] text-[#0969da] border border-[#54aeff]">Owner</span>}
                        </div>
                        <div className="text-xs text-[#656d76]">{c.role}</div>
                      </div>
                      <select className="w-[120px] px-2 py-1 border border-[#d0d7de] rounded-md text-sm bg-white" defaultValue={c.permission}>
                        <option>Admin</option>
                        <option>Write</option>
                        <option>Read</option>
                      </select>
                      {!c.isOwner && (
                        <button className="px-2 py-1 text-xs rounded-md border border-[#cf222e] text-[#cf222e] hover:bg-[#ffebe9]">削除</button>
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
              <div className="p-5">
                <ul className="m-0 p-0">
                  {webhooks.map((h) => (
                    <li key={h.url} className="px-4 py-3 border border-[#d0d7de] rounded-md mb-2 flex justify-between items-center">
                      <div>
                        <div className="font-mono text-[13px]">{h.url}</div>
                        <div className="text-xs text-[#656d76] mt-1">{h.meta}</div>
                      </div>
                      <div className="flex gap-2 items-center">
                        <span className={`text-xs px-2 py-0.5 rounded-full ${h.status === "Active" ? "bg-[#dafbe1] text-[#1a7f37]" : "bg-[#fff8c5] text-[#9a6700]"}`}>{h.status}</span>
                        <button className="px-2 py-1 text-xs rounded-md border border-[#d0d7de] bg-white hover:bg-[#f6f8fa]">編集</button>
                      </div>
                    </li>
                  ))}
                </ul>
              </div>
            </section>

            {/* Branch Protection */}
            <section className="bg-white border border-[#d0d7de] rounded-md overflow-hidden">
              <div className="px-5 py-4 border-b border-[#d0d7de] bg-[#f6f8fa] flex justify-between items-center">
                <div>
                  <h2 className="text-base font-semibold m-0">ブランチ保護ルール</h2>
                  <p className="text-[13px] text-[#656d76] mt-1">特定のブランチへのプッシュやマージに条件を設定します</p>
                </div>
                <button className="px-3 py-1.5 text-sm rounded-md bg-[#2da44e] text-white hover:bg-[#2c974b]">+ ルール追加</button>
              </div>
              <div className="p-5">
                <ul className="m-0 p-0">
                  <li className="px-4 py-3 border border-[#d0d7de] rounded-md mb-2 flex justify-between items-center">
                    <div>
                      <div className="font-mono font-semibold">main</div>
                      <div className="text-xs text-[#656d76] mt-1">PRレビュー必須 (2人) · ステータスチェック必須</div>
                    </div>
                    <button className="px-2 py-1 text-xs rounded-md border border-[#d0d7de] bg-white hover:bg-[#f6f8fa]">編集</button>
                  </li>
                  <li className="px-4 py-3 border border-[#d0d7de] rounded-md mb-2 flex justify-between items-center">
                    <div>
                      <div className="font-mono font-semibold">release/*</div>
                      <div className="text-xs text-[#656d76] mt-1">PRレビュー必須 (1人) · 強制プッシュ禁止</div>
                    </div>
                    <button className="px-2 py-1 text-xs rounded-md border border-[#d0d7de] bg-white hover:bg-[#f6f8fa]">編集</button>
                  </li>
                </ul>
              </div>
            </section>

            {/* Danger Zone */}
            <section className="bg-white border border-[#cf222e] rounded-md overflow-hidden">
              <div className="px-5 py-4 border-b border-[#d0d7de] bg-[#ffebe9] text-[#cf222e]">
                <h2 className="text-base font-semibold m-0">Danger Zone</h2>
                <p className="text-[13px] mt-1">取り扱い注意の操作です</p>
              </div>
              <div className="p-5">
                <div className="flex justify-between items-center py-4 border-b border-[#ffebe9]">
                  <div className="text-sm">
                    <div className="font-semibold text-[#cf222e]">可視性をPrivateに変更</div>
                    <div className="text-[#656d76] text-[13px] mt-0.5">公開を停止し、招待されたメンバーのみアクセス可能にします</div>
                  </div>
                  <button className="px-3 py-1.5 text-sm rounded-md border border-[#cf222e] text-[#cf222e] hover:bg-[#ffebe9]">変更</button>
                </div>
                <div className="flex justify-between items-center py-4 border-b border-[#ffebe9]">
                  <div className="text-sm">
                    <div className="font-semibold text-[#cf222e]">アーカイブ</div>
                    <div className="text-[#656d76] text-[13px] mt-0.5">読み取り専用にします</div>
                  </div>
                  <button className="px-3 py-1.5 text-sm rounded-md border border-[#cf222e] text-[#cf222e] hover:bg-[#ffebe9]">アーカイブ</button>
                </div>
                <div className="flex justify-between items-center py-4">
                  <div className="text-sm">
                    <div className="font-semibold text-[#cf222e]">リポジトリを削除</div>
                    <div className="text-[#656d76] text-[13px] mt-0.5">この操作は元に戻せません</div>
                  </div>
                  <button className="px-3 py-1.5 text-sm rounded-md bg-[#cf222e] text-white hover:bg-[#a40e26]">削除</button>
                </div>
              </div>
            </section>
          </main>
        </div>
      </div>
    </div>
  );
}

function Toggle({ label, on, onToggle, last }: { label: string; on: boolean; onToggle: () => void; last?: boolean }) {
  return (
    <div className={`grid grid-cols-[240px_1fr] gap-4 py-4 items-center ${last ? "" : "border-b border-[#eaeef2]"}`}>
      <div className="text-sm font-semibold">{label}</div>
      <button
        type="button"
        onClick={onToggle}
        className={`relative inline-block w-10 h-[22px] rounded-full transition-colors ${on ? "bg-[#2da44e]" : "bg-[#d0d7de]"}`}
        aria-pressed={on}
      >
        <span className={`absolute top-0.5 w-[18px] h-[18px] bg-white rounded-full transition-all ${on ? "left-5" : "left-0.5"}`} />
      </button>
    </div>
  );
}
