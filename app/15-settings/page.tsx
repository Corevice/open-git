"use client";

import Link from "next/link";
import { useState } from "react";

export default function SettingsPage() {
  const [profile, setProfile] = useState({
    displayName: "Taro Yamada",
    username: "taro_dev",
    email: "taro@example.com",
    bio: "OSS愛好家。Rust / TypeScript / Go。",
    location: "Tokyo, Japan",
  });

  const [tokenForm, setTokenForm] = useState({
    name: "",
    expiry: "30日",
    scopes: {
      repo: false,
      workflow: false,
      readUser: false,
      writePackages: false,
      adminOrg: false,
      deleteRepo: false,
    },
  });

  const [oauthForm, setOauthForm] = useState({
    name: "",
    homepage: "",
    description: "",
    callback: "",
  });

  const [sshForm, setSshForm] = useState({
    title: "",
    keyType: "Authentication Key",
    publicKey: "",
  });

  const tokens = [
    { name: "ci-deploy-token", status: "有効", badge: "bg-green-100 text-green-800", created: "2025-09-12", expires: "2025-12-11", lastUsed: "3時間前", scopes: ["repo", "workflow"], action: "取消" },
    { name: "local-dev", status: "期限間近", badge: "bg-yellow-100 text-yellow-800", created: "2025-07-01", expires: "2025-10-30", lastUsed: "昨日", scopes: ["read:user", "repo"], action: "取消" },
    { name: "old-script", status: "期限切れ", badge: "bg-red-100 text-red-800", created: "2024-12-01", expires: "2025-03-01", lastUsed: "7ヶ月前", scopes: [], action: "削除" },
  ];

  const oauthApps = [
    { name: "DevDashboard", clientId: "a7c3f9e2b1d4e5f6", created: "2025-08-20" },
    { name: "CodeReviewBot", clientId: "9f8e7d6c5b4a3210", created: "2025-06-10" },
  ];

  const sshKeys = [
    { title: "MacBook Pro 2024", type: "ED25519", fingerprint: "SHA256:aB3xKjF9pQrTvWxYz1234567890AbCdEfGhIjKlMn", added: "2025-09-01", lastUsed: "2時間前" },
    { title: "Work Desktop", type: "RSA 4096", fingerprint: "SHA256:zY9wVuT8sR7qPoN6mLkJ5hG4fE3dC2bA1zYxWvU", added: "2024-11-15", lastUsed: "5日前" },
  ];

  const handleSaveProfile = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  const handleCreateToken = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  const handleCreateOAuth = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  const handleAddSSH = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <header className="h-16 sticky top-0 z-50 bg-white/85 backdrop-blur border-b border-[#d0d7de] flex items-center justify-between px-6">
        <div className="flex items-center gap-2 font-extrabold text-lg">
          <span>🐙</span>
          <span>OpenHub</span>
        </div>
        <div className="flex items-center gap-4">
          <Link href="/04-dashboard" className="px-3 py-1.5 text-sm rounded-md hover:bg-gray-100">
            ダッシュボード
          </Link>
          <span className="px-2 py-1 text-xs rounded-full bg-blue-100 text-blue-800">@taro_dev</span>
        </div>
      </header>

      <div className="grid grid-cols-[240px_1fr] gap-8 max-w-[1280px] mx-auto p-6">
        <aside className="sticky top-6 self-start">
          <div className="pb-4 text-xs">
            <Link href="/04-dashboard" className="text-[#0969da] no-underline">
              ← ダッシュボードへ戻る
            </Link>
          </div>
          <div className="text-xs text-[#656d76] uppercase px-3 py-1 mb-2">設定</div>
          <nav className="flex flex-col gap-0.5">
            <a href="#profile" className="block px-3 py-2 rounded-md bg-gray-100 border-l-[3px] border-[#0969da] font-semibold">
              👤 プロファイル
            </a>
            <a href="#tokens" className="block px-3 py-2 rounded-md border-l-[3px] border-transparent hover:bg-gray-100">
              🔑 Personal Access Tokens
            </a>
            <a href="#oauth" className="block px-3 py-2 rounded-md border-l-[3px] border-transparent hover:bg-gray-100">
              🔗 OAuth Apps
            </a>
            <a href="#ssh" className="block px-3 py-2 rounded-md border-l-[3px] border-transparent hover:bg-gray-100">
              🖥️ SSH Keys
            </a>
            <a href="#notifications" className="block px-3 py-2 rounded-md border-l-[3px] border-transparent hover:bg-gray-100">
              🔔 通知
            </a>
            <a href="#billing" className="block px-3 py-2 rounded-md border-l-[3px] border-transparent hover:bg-gray-100">
              💳 プラン
            </a>
            <a href="#danger" className="block px-3 py-2 rounded-md border-l-[3px] border-transparent hover:bg-gray-100 text-[#cf222e]">
              ⚠️ 危険な操作
            </a>
          </nav>
        </aside>

        <main className="min-w-0">
          {/* Profile */}
          <section id="profile" className="bg-white border border-[#d0d7de] rounded-lg mb-6">
            <div className="px-5 py-4 border-b border-[#d0d7de]">
              <h2 className="text-lg font-semibold m-0">プロファイル</h2>
              <p className="text-xs text-[#656d76] m-0 mt-1">あなたの公開プロフィール情報を編集します。</p>
            </div>
            <form onSubmit={handleSaveProfile}>
              <div className="px-5 py-2">
                <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[#f0f1f3]">
                  <label className="font-medium text-sm">アバター</label>
                  <div className="flex items-center gap-4">
                    <div className="w-16 h-16 rounded-full bg-gradient-to-br from-[#0969da] to-[#8250df] flex items-center justify-center text-white text-2xl font-semibold">
                      T
                    </div>
                    <button type="button" className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md hover:bg-gray-50">
                      画像を変更
                    </button>
                  </div>
                </div>
                <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[#f0f1f3]">
                  <label className="font-medium text-sm">表示名</label>
                  <input
                    type="text"
                    value={profile.displayName}
                    onChange={(e) => setProfile({ ...profile, displayName: e.target.value })}
                    className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm"
                  />
                </div>
                <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[#f0f1f3]">
                  <label className="font-medium text-sm">ユーザー名</label>
                  <input
                    type="text"
                    value={profile.username}
                    onChange={(e) => setProfile({ ...profile, username: e.target.value })}
                    className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm font-mono"
                  />
                </div>
                <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[#f0f1f3]">
                  <label className="font-medium text-sm">メールアドレス</label>
                  <input
                    type="email"
                    value={profile.email}
                    onChange={(e) => setProfile({ ...profile, email: e.target.value })}
                    className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm"
                  />
                </div>
                <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[#f0f1f3]">
                  <label className="font-medium text-sm">自己紹介</label>
                  <textarea
                    value={profile.bio}
                    onChange={(e) => setProfile({ ...profile, bio: e.target.value })}
                    className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm font-mono min-h-[100px]"
                  />
                </div>
                <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-center">
                  <label className="font-medium text-sm">所在地</label>
                  <input
                    type="text"
                    value={profile.location}
                    onChange={(e) => setProfile({ ...profile, location: e.target.value })}
                    className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm"
                  />
                </div>
              </div>
              <div className="px-5 py-4 border-t border-[#d0d7de] bg-[#f6f8fa] rounded-b-lg flex justify-end gap-2">
                <button type="button" className="px-3 py-1.5 text-sm rounded-md hover:bg-gray-100">
                  キャンセル
                </button>
                <button type="submit" className="px-3 py-1.5 text-sm rounded-md bg-[#0969da] text-white hover:bg-[#0860c4]">
                  変更を保存
                </button>
              </div>
            </form>
          </section>

          {/* Tokens */}
          <section id="tokens" className="bg-white border border-[#d0d7de] rounded-lg mb-6">
            <div className="px-5 py-4 border-b border-[#d0d7de]">
              <h2 className="text-lg font-semibold m-0">Personal Access Tokens (PAT)</h2>
              <p className="text-xs text-[#656d76] m-0 mt-1">APIアクセス用のトークンを発行・管理します。トークンは発行時にのみ表示されます。</p>
            </div>
            <form onSubmit={handleCreateToken}>
              <div className="px-5 py-2">
                <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[#f0f1f3]">
                  <label className="font-medium text-sm">
                    トークン名 <span className="text-[#cf222e]">*</span>
                  </label>
                  <input
                    type="text"
                    value={tokenForm.name}
                    onChange={(e) => setTokenForm({ ...tokenForm, name: e.target.value })}
                    placeholder="例: ci-deploy-token"
                    className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm"
                  />
                </div>
                <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[#f0f1f3]">
                  <label className="font-medium text-sm">有効期限</label>
                  <select
                    value={tokenForm.expiry}
                    onChange={(e) => setTokenForm({ ...tokenForm, expiry: e.target.value })}
                    className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm"
                  >
                    <option>30日</option>
                    <option>60日</option>
                    <option>90日</option>
                    <option>1年</option>
                    <option>無期限（非推奨）</option>
                  </select>
                </div>
                <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-start">
                  <label className="font-medium text-sm pt-1">スコープ</label>
                  <div className="grid grid-cols-2 gap-2 text-xs">
                    <label className="flex items-center gap-1.5 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={tokenForm.scopes.repo}
                        onChange={(e) => setTokenForm({ ...tokenForm, scopes: { ...tokenForm.scopes, repo: e.target.checked } })}
                      />
                      repo (リポジトリ全権限)
                    </label>
                    <label className="flex items-center gap-1.5 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={tokenForm.scopes.workflow}
                        onChange={(e) => setTokenForm({ ...tokenForm, scopes: { ...tokenForm.scopes, workflow: e.target.checked } })}
                      />
                      workflow
                    </label>
                    <label className="flex items-center gap-1.5 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={tokenForm.scopes.readUser}
                        onChange={(e) => setTokenForm({ ...tokenForm, scopes: { ...tokenForm.scopes, readUser: e.target.checked } })}
                      />
                      read:user
                    </label>
                    <label className="flex items-center gap-1.5 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={tokenForm.scopes.writePackages}
                        onChange={(e) => setTokenForm({ ...tokenForm, scopes: { ...tokenForm.scopes, writePackages: e.target.checked } })}
                      />
                      write:packages
                    </label>
                    <label className="flex items-center gap-1.5 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={tokenForm.scopes.adminOrg}
                        onChange={(e) => setTokenForm({ ...tokenForm, scopes: { ...tokenForm.scopes, adminOrg: e.target.checked } })}
                      />
                      admin:org
                    </label>
                    <label className="flex items-center gap-1.5 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={tokenForm.scopes.deleteRepo}
                        onChange={(e) => setTokenForm({ ...tokenForm, scopes: { ...tokenForm.scopes, deleteRepo: e.target.checked } })}
                      />
                      delete_repo
                    </label>
                  </div>
                </div>
              </div>
              <div className="px-5 py-4 border-t border-[#d0d7de] bg-[#f6f8fa] flex justify-end">
                <button type="submit" className="px-3 py-1.5 text-sm rounded-md bg-[#0969da] text-white hover:bg-[#0860c4]">
                  トークンを発行
                </button>
              </div>
            </form>

            <div className="px-5 py-4 border-t border-[#d0d7de]">
              <h3 className="text-base font-semibold m-0">発行済みトークン</h3>
            </div>
            <div>
              {tokens.map((t) => (
                <div key={t.name} className="flex justify-between items-center px-5 py-3.5 border-b border-[#f0f1f3] last:border-b-0">
                  <div>
                    <div className="text-sm font-semibold">
                      {t.name} <span className={`ml-2 px-2 py-0.5 text-xs rounded-full ${t.badge}`}>{t.status}</span>
                    </div>
                    <div className="text-xs text-[#656d76] mt-1">
                      作成: {t.created} / 期限: {t.expires} / 最終使用: {t.lastUsed}
                    </div>
                    {t.scopes.length > 0 && (
                      <div className="flex gap-1 mt-1.5 flex-wrap">
                        {t.scopes.map((s) => (
                          <span key={s} className="px-1.5 py-0.5 text-xs bg-gray-100 rounded">
                            {s}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                  <button className={`px-2.5 py-1 text-xs rounded-md ${t.action === "取消" ? "bg-[#cf222e] text-white hover:bg-[#a40e26]" : "border border-[#d0d7de] hover:bg-gray-50"}`}>
                    {t.action}
                  </button>
                </div>
              ))}
            </div>
          </section>

          {/* OAuth */}
          <section id="oauth" className="bg-white border border-[#d0d7de] rounded-lg mb-6">
            <div className="px-5 py-4 border-b border-[#d0d7de]">
              <h2 className="text-lg font-semibold m-0">OAuth Apps</h2>
              <p className="text-xs text-[#656d76] m-0 mt-1">外部アプリケーションがあなたのアカウントへアクセスするためのOAuthアプリを登録します。</p>
            </div>
            <form onSubmit={handleCreateOAuth}>
              <div className="px-5 py-2">
                <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[#f0f1f3]">
                  <label className="font-medium text-sm">
                    アプリ名 <span className="text-[#cf222e]">*</span>
                  </label>
                  <input
                    type="text"
                    value={oauthForm.name}
                    onChange={(e) => setOauthForm({ ...oauthForm, name: e.target.value })}
                    placeholder="My Awesome App"
                    className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm"
                  />
                </div>
                <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[#f0f1f3]">
                  <label className="font-medium text-sm">
                    ホームページURL <span className="text-[#cf222e]">*</span>
                  </label>
                  <input
                    type="text"
                    value={oauthForm.homepage}
                    onChange={(e) => setOauthForm({ ...oauthForm, homepage: e.target.value })}
                    placeholder="https://example.com"
                    className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm"
                  />
                </div>
                <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[#f0f1f3]">
                  <label className="font-medium text-sm">説明</label>
                  <input
                    type="text"
                    value={oauthForm.description}
                    onChange={(e) => setOauthForm({ ...oauthForm, description: e.target.value })}
                    placeholder="アプリの説明（任意）"
                    className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm"
                  />
                </div>
                <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-center">
                  <label className="font-medium text-sm">
                    コールバックURL <span className="text-[#cf222e]">*</span>
                  </label>
                  <input
                    type="text"
                    value={oauthForm.callback}
                    onChange={(e) => setOauthForm({ ...oauthForm, callback: e.target.value })}
                    placeholder="https://example.com/auth/callback"
                    className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm"
                  />
                </div>
              </div>
              <div className="px-5 py-4 border-t border-[#d0d7de] bg-[#f6f8fa] flex justify-end">
                <button type="submit" className="px-3 py-1.5 text-sm rounded-md bg-[#0969da] text-white hover:bg-[#0860c4]">
                  OAuth Appを作成
                </button>
              </div>
            </form>

            <div className="px-5 py-4 border-t border-[#d0d7de]">
              <h3 className="text-base font-semibold m-0">登録済みアプリ</h3>
            </div>
            <div>
              {oauthApps.map((a) => (
                <div key={a.clientId} className="flex justify-between items-center px-5 py-3.5 border-b border-[#f0f1f3] last:border-b-0">
                  <div>
                    <div className="text-sm font-semibold">{a.name}</div>
                    <div className="text-xs text-[#656d76] mt-1 font-mono">Client ID: {a.clientId}</div>
                    <div className="text-xs text-[#656d76]">作成: {a.created}</div>
                  </div>
                  <div className="flex gap-2">
                    <button className="px-2.5 py-1 text-xs rounded-md border border-[#d0d7de] hover:bg-gray-50">編集</button>
                    <button className="px-2.5 py-1 text-xs rounded-md bg-[#cf222e] text-white hover:bg-[#a40e26]">削除</button>
                  </div>
                </div>
              ))}
            </div>
          </section>

          {/* SSH */}
          <section id="ssh" className="bg-white border border-[#d0d7de] rounded-lg mb-6">
            <div className="px-5 py-4 border-b border-[#d0d7de]">
              <h2 className="text-lg font-semibold m-0">SSH Keys</h2>
              <p className="text-xs text-[#656d76] m-0 mt-1">Gitリポジトリへの SSH 認証に使用する公開鍵を追加します。</p>
            </div>
            <form onSubmit={handleAddSSH}>
              <div className="px-5 py-2">
                <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[#f0f1f3]">
                  <label className="font-medium text-sm">
                    タイトル <span className="text-[#cf222e]">*</span>
                  </label>
                  <input
                    type="text"
                    value={sshForm.title}
                    onChange={(e) => setSshForm({ ...sshForm, title: e.target.value })}
                    placeholder="例: MacBook Pro 2024"
                    className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm"
                  />
                </div>
                <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[#f0f1f3]">
                  <label className="font-medium text-sm">キータイプ</label>
                  <select
                    value={sshForm.keyType}
                    onChange={(e) => setSshForm({ ...sshForm, keyType: e.target.value })}
                    className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm"
                  >
                    <option>Authentication Key</option>
                    <option>Signing Key</option>
                  </select>
                </div>
                <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-center">
                  <label className="font-medium text-sm">
                    公開鍵 <span className="text-[#cf222e]">*</span>
                  </label>
                  <textarea
                    value={sshForm.publicKey}
                    onChange={(e) => setSshForm({ ...sshForm, publicKey: e.target.value })}
                    placeholder="ssh-ed25519 AAAAC3Nz... your_email@example.com"
                    className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm font-mono min-h-[100px]"
                  />
                </div>
              </div>
              <div className="px-5 py-4 border-t border-[#d0d7de] bg-[#f6f8fa] flex justify-end">
                <button type="submit" className="px-3 py-1.5 text-sm rounded-md bg-[#0969da] text-white hover:bg-[#0860c4]">
                  SSHキーを追加
                </button>
              </div>
            </form>

            <div className="px-5 py-4 border-t border-[#d0d7de]">
              <h3 className="text-base font-semibold m-0">登録済みSSHキー</h3>
            </div>
            <div>
              {sshKeys.map((k) => (
                <div key={k.fingerprint} className="flex justify-between items-center px-5 py-3.5 border-b border-[#f0f1f3] last:border-b-0">
                  <div>
                    <div className="text-sm font-semibold">
                      {k.title} <span className="ml-2 px-2 py-0.5 text-xs rounded-full bg-blue-100 text-blue-800">{k.type}</span>
                    </div>
                    <div className="text-xs text-[#656d76] mt-1 font-mono">{k.fingerprint}</div>
                    <div className="text-xs text-[#656d76]">追加: {k.added} / 最終使用: {k.lastUsed}</div>
                  </div>
                  <button className="px-2.5 py-1 text-xs rounded-md bg-[#cf222e] text-white hover:bg-[#a40e26]">削除</button>
                </div>
              ))}
            </div>
          </section>

          {/* Danger Zone */}
          <section id="danger" className="bg-white border border-[#cf222e] rounded-lg mb-6">
            <div className="px-5 py-4 border-b border-[#cf222e] bg-[#ffebe9] rounded-t-lg">
              <h2 className="text-lg font-semibold m-0 text-[#cf222e]">⚠️ 危険な操作</h2>
              <p className="text-xs text-[#cf222e] m-0 mt-1">これらの操作は元に戻せません。慎重に実行してください。</p>
            </div>
            <div className="px-5 py-2">
              <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[#f0f1f3]">
                <label className="font-medium text-sm">全トークンを無効化</label>
                <div>
                  <p className="text-xs text-[#cf222e] mb-2">発行済みの全PATが即座に取り消されます。</p>
                  <button className="px-2.5 py-1 text-xs rounded-md bg-[#cf222e] text-white hover:bg-[#a40e26]">
                    全トークンを無効化
                  </button>
                </div>
              </div>
              <div className="grid grid-cols-[200px_1fr] gap-4 py-3 items-center">
                <label className="font-medium text-sm">アカウント削除</label>
                <div>
                  <p className="text-xs text-[#cf222e] mb-2">あなたのアカウントと全データが完全に削除されます。</p>
                  <button className="px-2.5 py-1 text-xs rounded-md bg-[#cf222e] text-white hover:bg-[#a40e26]">
                    アカウントを削除
                  </button>
                </div>
              </div>
            </div>
          </section>

          <div className="text-center py-4 text-xs text-[#656d76]">
            <Link href="/04-dashboard" className="text-[#0969da] no-underline">
              ← ダッシュボードへ戻る
            </Link>
          </div>
        </main>
      </div>
    </div>
  );
}
