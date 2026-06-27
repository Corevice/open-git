"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { ApiClient } from "@/lib/api";
import type { SSHKey } from "@/lib/api-types";

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
    {
      name: "ci-deploy-token",
      status: "有効",
      statusClass: "bg-green-100 text-green-800",
      meta: "作成: 2025-09-12 / 期限: 2025-12-11 / 最終使用: 3時間前",
      scopes: ["repo", "workflow"],
      action: "取消",
      actionDanger: true,
    },
    {
      name: "local-dev",
      status: "期限間近",
      statusClass: "bg-yellow-100 text-yellow-800",
      meta: "作成: 2025-07-01 / 期限: 2025-10-30 / 最終使用: 昨日",
      scopes: ["read:user", "repo"],
      action: "取消",
      actionDanger: true,
    },
    {
      name: "old-script",
      status: "期限切れ",
      statusClass: "bg-red-100 text-red-800",
      meta: "作成: 2024-12-01 / 期限: 2025-03-01 / 最終使用: 7ヶ月前",
      scopes: [],
      action: "削除",
      actionDanger: false,
    },
  ];

  const oauthApps = [
    { name: "DevDashboard", clientId: "a7c3f9e2b1d4e5f6", created: "2025-08-20" },
    { name: "CodeReviewBot", clientId: "9f8e7d6c5b4a3210", created: "2025-06-10" },
  ];

  const [sshKeys, setSshKeys] = useState<SSHKey[]>([]);
  const apiClient = new ApiClient(process.env.NEXT_PUBLIC_API_URL ?? "");

  useEffect(() => {
    apiClient.sshKeys.list().then(setSshKeys).catch(console.error);
  }, []);

  const handleSaveProfile = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  const handleCreateToken = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  const handleCreateOauth = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  const handleAddSsh = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const key = await apiClient.sshKeys.create(sshForm.title, sshForm.publicKey);
      setSshKeys((prev) => [...prev, key]);
      setSshForm({ title: "", keyType: "Authentication Key", publicKey: "" });
    } catch (err) {
      alert((err as Error).message);
    }
  };

  const handleDeleteSsh = async (id: string) => {
    await apiClient.sshKeys.remove(id);
    setSshKeys((prev) => prev.filter((k) => k.id !== id));
  };

  return (
    <div className="min-h-screen bg-[color:var(--bg-base)] text-[color:var(--text-primary)]">
      <header className="h-16 sticky top-0 z-50 bg-white/85 backdrop-blur border-b border-[color:var(--border)] flex items-center justify-between px-6">
        <div className="flex items-center gap-2 font-extrabold text-lg">
          <span>🐙</span>
          <span>OpenHub</span>
        </div>
        <div className="flex items-center gap-3">
          <Link href="/04-dashboard" className="px-3 py-1.5 text-sm rounded-md hover:bg-[color:var(--bg-muted)]">
            ダッシュボード
          </Link>
          <span className="px-2.5 py-1 text-xs rounded-full bg-[color:var(--info-light)] text-[color:var(--info)]">
            @taro_dev
          </span>
        </div>
      </header>

      <div className="grid grid-cols-1 md:grid-cols-[240px_1fr] gap-8 max-w-[1280px] mx-auto px-6 py-6">
        <aside className="md:sticky md:top-20 self-start">
          <div className="text-xs text-[color:var(--text-secondary)] pb-4">
            <Link href="/04-dashboard" className="text-[color:var(--primary)]">
              ← ダッシュボードへ戻る
            </Link>
          </div>
          <div className="text-xs uppercase text-[color:var(--text-muted)] px-3 mb-2">設定</div>
          <nav className="flex flex-col gap-0.5">
            <a href="#profile" className="block px-3 py-2 rounded-md border-l-[3px] border-[color:var(--primary)] bg-[color:var(--bg-muted)] font-semibold text-sm">
              👤 プロファイル
            </a>
            <a href="#tokens" className="block px-3 py-2 rounded-md border-l-[3px] border-transparent hover:bg-[color:var(--bg-muted)] text-sm">
              🔑 Personal Access Tokens
            </a>
            <a href="#oauth" className="block px-3 py-2 rounded-md border-l-[3px] border-transparent hover:bg-[color:var(--bg-muted)] text-sm">
              🔗 OAuth Apps
            </a>
            <a href="#ssh" className="block px-3 py-2 rounded-md border-l-[3px] border-transparent hover:bg-[color:var(--bg-muted)] text-sm">
              🖥️ SSH Keys
            </a>
            <a href="#notifications" className="block px-3 py-2 rounded-md border-l-[3px] border-transparent hover:bg-[color:var(--bg-muted)] text-sm">
              🔔 通知
            </a>
            <a href="#billing" className="block px-3 py-2 rounded-md border-l-[3px] border-transparent hover:bg-[color:var(--bg-muted)] text-sm">
              💳 プラン
            </a>
            <a href="#danger" className="block px-3 py-2 rounded-md border-l-[3px] border-transparent hover:bg-[color:var(--bg-muted)] text-sm text-[color:var(--danger)]">
              ⚠️ 危険な操作
            </a>
          </nav>
        </aside>

        <main className="min-w-0">
          {/* Profile */}
          <form onSubmit={handleSaveProfile} id="profile" className="bg-white border border-[color:var(--border)] rounded-lg mb-6">
            <div className="px-5 py-4 border-b border-[color:var(--border)]">
              <h2 className="text-lg font-bold mb-1">プロファイル</h2>
              <p className="text-xs text-[color:var(--text-secondary)]">あなたの公開プロフィール情報を編集します。</p>
            </div>
            <div className="px-5 py-3">
              <div className="grid grid-cols-1 md:grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[color:var(--border-subtle)]">
                <label className="text-sm font-medium">アバター</label>
                <div className="flex items-center gap-4">
                  <div className="w-16 h-16 rounded-full bg-gradient-to-br from-[color:var(--primary)] to-[color:var(--secondary)] flex items-center justify-center text-white text-2xl font-semibold">
                    T
                  </div>
                  <button type="button" className="px-3 py-1.5 text-sm border border-[color:var(--border-strong)] rounded-md hover:bg-[color:var(--bg-muted)]">
                    画像を変更
                  </button>
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[color:var(--border-subtle)]">
                <label className="text-sm font-medium">表示名</label>
                <input
                  type="text"
                  value={profile.displayName}
                  onChange={(e) => setProfile({ ...profile, displayName: e.target.value })}
                  className="w-full px-3 py-2 border border-[color:var(--border-strong)] rounded-md text-sm"
                />
              </div>
              <div className="grid grid-cols-1 md:grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[color:var(--border-subtle)]">
                <label className="text-sm font-medium">ユーザー名</label>
                <input
                  type="text"
                  value={profile.username}
                  onChange={(e) => setProfile({ ...profile, username: e.target.value })}
                  className="w-full px-3 py-2 border border-[color:var(--border-strong)] rounded-md text-sm font-mono"
                />
              </div>
              <div className="grid grid-cols-1 md:grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[color:var(--border-subtle)]">
                <label className="text-sm font-medium">メールアドレス</label>
                <input
                  type="email"
                  value={profile.email}
                  onChange={(e) => setProfile({ ...profile, email: e.target.value })}
                  className="w-full px-3 py-2 border border-[color:var(--border-strong)] rounded-md text-sm"
                />
              </div>
              <div className="grid grid-cols-1 md:grid-cols-[200px_1fr] gap-4 py-3 items-start border-b border-[color:var(--border-subtle)]">
                <label className="text-sm font-medium pt-2">自己紹介</label>
                <textarea
                  value={profile.bio}
                  onChange={(e) => setProfile({ ...profile, bio: e.target.value })}
                  className="w-full px-3 py-2 border border-[color:var(--border-strong)] rounded-md text-sm font-mono min-h-[100px]"
                />
              </div>
              <div className="grid grid-cols-1 md:grid-cols-[200px_1fr] gap-4 py-3 items-center">
                <label className="text-sm font-medium">所在地</label>
                <input
                  type="text"
                  value={profile.location}
                  onChange={(e) => setProfile({ ...profile, location: e.target.value })}
                  className="w-full px-3 py-2 border border-[color:var(--border-strong)] rounded-md text-sm"
                />
              </div>
            </div>
            <div className="px-5 py-4 border-t border-[color:var(--border)] bg-[color:var(--bg-base)] rounded-b-lg flex justify-end gap-2">
              <button type="button" className="px-3 py-1.5 text-sm rounded-md hover:bg-[color:var(--bg-muted)]">
                キャンセル
              </button>
              <button type="submit" className="px-3 py-1.5 text-sm rounded-md bg-[color:var(--primary)] text-white hover:bg-[color:var(--primary-hover)]">
                変更を保存
              </button>
            </div>
          </form>

          {/* Tokens */}
          <section id="tokens" className="bg-white border border-[color:var(--border)] rounded-lg mb-6">
            <div className="px-5 py-4 border-b border-[color:var(--border)]">
              <h2 className="text-lg font-bold mb-1">Personal Access Tokens (PAT)</h2>
              <p className="text-xs text-[color:var(--text-secondary)]">APIアクセス用のトークンを発行・管理します。トークンは発行時にのみ表示されます。</p>
            </div>
            <form onSubmit={handleCreateToken}>
              <div className="px-5 py-3">
                <div className="grid grid-cols-1 md:grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[color:var(--border-subtle)]">
                  <label className="text-sm font-medium">トークン名 <span className="text-[color:var(--danger)]">*</span></label>
                  <input
                    type="text"
                    placeholder="例: ci-deploy-token"
                    value={tokenForm.name}
                    onChange={(e) => setTokenForm({ ...tokenForm, name: e.target.value })}
                    className="w-full px-3 py-2 border border-[color:var(--border-strong)] rounded-md text-sm"
                  />
                </div>
                <div className="grid grid-cols-1 md:grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[color:var(--border-subtle)]">
                  <label className="text-sm font-medium">有効期限</label>
                  <select
                    value={tokenForm.expiry}
                    onChange={(e) => setTokenForm({ ...tokenForm, expiry: e.target.value })}
                    className="w-full px-3 py-2 border border-[color:var(--border-strong)] rounded-md text-sm"
                  >
                    <option>30日</option>
                    <option>60日</option>
                    <option>90日</option>
                    <option>1年</option>
                    <option>無期限（非推奨）</option>
                  </select>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-[200px_1fr] gap-4 py-3 items-start">
                  <label className="text-sm font-medium pt-1">スコープ</label>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-sm">
                    {([
                      ["repo", "repo (リポジトリ全権限)"],
                      ["workflow", "workflow"],
                      ["readUser", "read:user"],
                      ["writePackages", "write:packages"],
                      ["adminOrg", "admin:org"],
                      ["deleteRepo", "delete_repo"],
                    ] as const).map(([key, label]) => (
                      <label key={key} className="flex items-center gap-1.5 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={tokenForm.scopes[key]}
                          onChange={(e) =>
                            setTokenForm({
                              ...tokenForm,
                              scopes: { ...tokenForm.scopes, [key]: e.target.checked },
                            })
                          }
                        />
                        {label}
                      </label>
                    ))}
                  </div>
                </div>
              </div>
              <div className="px-5 py-4 border-t border-[color:var(--border)] bg-[color:var(--bg-base)] flex justify-end gap-2">
                <button type="submit" className="px-3 py-1.5 text-sm rounded-md bg-[color:var(--primary)] text-white hover:bg-[color:var(--primary-hover)]">
                  トークンを発行
                </button>
              </div>
            </form>

            <div className="px-5 py-4 border-t border-[color:var(--border)]">
              <h3 className="text-sm font-bold">発行済みトークン</h3>
            </div>
            <div>
              {tokens.map((t) => (
                <div key={t.name} className="flex justify-between items-center px-5 py-3.5 border-b border-[color:var(--border-subtle)] last:border-b-0">
                  <div>
                    <div className="text-sm font-semibold flex items-center gap-2">
                      {t.name}
                      <span className={`px-2 py-0.5 text-xs rounded-full ${t.statusClass}`}>{t.status}</span>
                    </div>
                    <div className="text-xs text-[color:var(--text-secondary)] mt-1">{t.meta}</div>
                    {t.scopes.length > 0 && (
                      <div className="flex gap-1 mt-1.5 flex-wrap">
                        {t.scopes.map((s) => (
                          <span key={s} className="px-2 py-0.5 text-xs rounded bg-[color:var(--bg-muted)] text-[color:var(--text-secondary)] font-mono">
                            {s}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                  <button
                    type="button"
                    className={
                      t.actionDanger
                        ? "px-3 py-1 text-xs rounded-md bg-[color:var(--danger)] text-white hover:opacity-90"
                        : "px-3 py-1 text-xs rounded-md border border-[color:var(--border-strong)] hover:bg-[color:var(--bg-muted)]"
                    }
                  >
                    {t.action}
                  </button>
                </div>
              ))}
            </div>
          </section>

          {/* OAuth */}
          <section id="oauth" className="bg-white border border-[color:var(--border)] rounded-lg mb-6">
            <div className="px-5 py-4 border-b border-[color:var(--border)]">
              <h2 className="text-lg font-bold mb-1">OAuth Apps</h2>
              <p className="text-xs text-[color:var(--text-secondary)]">外部アプリケーションがあなたのアカウントへアクセスするためのOAuthアプリを登録します。</p>
            </div>
            <form onSubmit={handleCreateOauth}>
              <div className="px-5 py-3">
                {([
                  ["name", "アプリ名", "My Awesome App", true],
                  ["homepage", "ホームページURL", "https://example.com", true],
                  ["description", "説明", "アプリの説明（任意）", false],
                  ["callback", "コールバックURL", "https://example.com/auth/callback", true],
                ] as const).map(([key, label, placeholder, required]) => (
                  <div key={key} className="grid grid-cols-1 md:grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[color:var(--border-subtle)] last:border-b-0">
                    <label className="text-sm font-medium">
                      {label} {required && <span className="text-[color:var(--danger)]">*</span>}
                    </label>
                    <input
                      type="text"
                      placeholder={placeholder}
                      value={oauthForm[key]}
                      onChange={(e) => setOauthForm({ ...oauthForm, [key]: e.target.value })}
                      className="w-full px-3 py-2 border border-[color:var(--border-strong)] rounded-md text-sm"
                    />
                  </div>
                ))}
              </div>
              <div className="px-5 py-4 border-t border-[color:var(--border)] bg-[color:var(--bg-base)] flex justify-end gap-2">
                <button type="submit" className="px-3 py-1.5 text-sm rounded-md bg-[color:var(--primary)] text-white hover:bg-[color:var(--primary-hover)]">
                  OAuth Appを作成
                </button>
              </div>
            </form>

            <div className="px-5 py-4 border-t border-[color:var(--border)]">
              <h3 className="text-sm font-bold">登録済みアプリ</h3>
            </div>
            <div>
              {oauthApps.map((app) => (
                <div key={app.clientId} className="flex justify-between items-center px-5 py-3.5 border-b border-[color:var(--border-subtle)] last:border-b-0">
                  <div>
                    <div className="text-sm font-semibold">{app.name}</div>
                    <div className="text-xs text-[color:var(--text-secondary)] mt-1 font-mono">Client ID: {app.clientId}</div>
                    <div className="text-xs text-[color:var(--text-secondary)]">作成: {app.created}</div>
                  </div>
                  <div className="flex gap-2">
                    <button type="button" className="px-3 py-1 text-xs rounded-md border border-[color:var(--border-strong)] hover:bg-[color:var(--bg-muted)]">
                      編集
                    </button>
                    <button type="button" className="px-3 py-1 text-xs rounded-md bg-[color:var(--danger)] text-white hover:opacity-90">
                      削除
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </section>

          {/* SSH */}
          <section id="ssh" className="bg-white border border-[color:var(--border)] rounded-lg mb-6">
            <div className="px-5 py-4 border-b border-[color:var(--border)]">
              <h2 className="text-lg font-bold mb-1">SSH Keys</h2>
              <p className="text-xs text-[color:var(--text-secondary)]">Gitリポジトリへの SSH 認証に使用する公開鍵を追加します。</p>
            </div>
            <form onSubmit={handleAddSsh}>
              <div className="px-5 py-3">
                <div className="grid grid-cols-1 md:grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[color:var(--border-subtle)]">
                  <label className="text-sm font-medium">タイトル <span className="text-[color:var(--danger)]">*</span></label>
                  <input
                    type="text"
                    placeholder="例: MacBook Pro 2024"
                    value={sshForm.title}
                    onChange={(e) => setSshForm({ ...sshForm, title: e.target.value })}
                    className="w-full px-3 py-2 border border-[color:var(--border-strong)] rounded-md text-sm"
                  />
                </div>
                <div className="grid grid-cols-1 md:grid-cols-[200px_1fr] gap-4 py-3 items-center border-b border-[color:var(--border-subtle)]">
                  <label className="text-sm font-medium">キータイプ</label>
                  <select
                    value={sshForm.keyType}
                    onChange={(e) => setSshForm({ ...sshForm, keyType: e.target.value })}
                    className="w-full px-3 py-2 border border-[color:var(--border-strong)] rounded-md text-sm"
                  >
                    <option>Authentication Key</option>
                    <option>Signing Key</option>
                  </select>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-[200px_1fr] gap-4 py-3 items-start">
                  <label className="text-sm font-medium pt-2">公開鍵 <span className="text-[color:var(--danger)]">*</span></label>
                  <textarea
                    placeholder="ssh-ed25519 AAAAC3Nz... your_email@example.com"
                    value={sshForm.publicKey}
                    onChange={(e) => setSshForm({ ...sshForm, publicKey: e.target.value })}
                    className="w-full px-3 py-2 border border-[color:var(--border-strong)] rounded-md text-sm font-mono min-h-[100px]"
                  />
                </div>
              </div>
              <div className="px-5 py-4 border-t border-[color:var(--border)] bg-[color:var(--bg-base)] flex justify-end gap-2">
                <button type="submit" className="px-3 py-1.5 text-sm rounded-md bg-[color:var(--primary)] text-white hover:bg-[color:var(--primary-hover)]">
                  SSHキーを追加
                </button>
              </div>
            </form>

            <div className="px-5 py-4 border-t border-[color:var(--border)]">
              <h3 className="text-sm font-bold">登録済みSSHキー</h3>
            </div>
            <div>
              {sshKeys.map((k) => (
                <div key={k.id} className="flex justify-between items-center px-5 py-3.5 border-b border-[color:var(--border-subtle)] last:border-b-0">
                  <div>
                    <div className="text-sm font-semibold flex items-center gap-2">
                      {k.title}
                      <span className="px-2 py-0.5 text-xs rounded-full bg-[color:var(--primary-light)] text-[color:var(--primary)]">
                        {k.key_type}
                      </span>
                    </div>
                    <div className="text-xs text-[color:var(--text-secondary)] mt-1 font-mono break-all">{k.fingerprint}</div>
                    <div className="text-xs text-[color:var(--text-secondary)]">
                      追加: {k.created_at} / 最終使用: {k.last_used_at ?? "未使用"}
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={() => handleDeleteSsh(k.id)}
                    className="px-3 py-1 text-xs rounded-md bg-[color:var(--danger)] text-white hover:opacity-90 shrink-0 ml-3"
                  >
                    削除
                  </button>
                </div>
              ))}
            </div>
          </section>

          {/* Danger Zone */}
          <section id="danger" className="bg-white border border-[color:var(--danger)] rounded-lg mb-6">
            <div className="px-5 py-4 border-b border-[color:var(--danger)] bg-[color:var(--danger-light)] rounded-t-lg">
              <h2 className="text-lg font-bold mb-1 text-[color:var(--danger)]">⚠️ 危険な操作</h2>
              <p className="text-xs text-[color:var(--danger)]">これらの操作は元に戻せません。慎重に実行してください。</p>
            </div>
            <div className="px-5 py-3">
              <div className="grid grid-cols-1 md:grid-cols-[200px_1fr] gap-4 py-3 items-start border-b border-[color:var(--border-subtle)]">
                <label className="text-sm font-medium">全トークンを無効化</label>
                <div>
                  <p className="text-xs text-[color:var(--danger)] mb-2">発行済みの全PATが即座に取り消されます。</p>
                  <button type="button" className="px-3 py-1.5 text-sm rounded-md bg-[color:var(--danger)] text-white hover:opacity-90">
                    全トークンを無効化
                  </button>
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-[200px_1fr] gap-4 py-3 items-start">
                <label className="text-sm font-medium">アカウント削除</label>
                <div>
                  <p className="text-xs text-[color:var(--danger)] mb-2">あなたのアカウントと全データが完全に削除されます。</p>
                  <button type="button" className="px-3 py-1.5 text-sm rounded-md bg-[color:var(--danger)] text-white hover:opacity-90">
                    アカウントを削除
                  </button>
                </div>
              </div>
            </div>
          </section>

          <div className="text-center py-4 text-xs text-[color:var(--text-secondary)]">
            <Link href="/04-dashboard" className="text-[color:var(--primary)]">
              ← ダッシュボードへ戻る
            </Link>
          </div>
        </main>
      </div>
    </div>
  );
}
