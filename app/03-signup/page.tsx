"use client";

import Link from "next/link";
import { useState } from "react";

export default function SignupPage() {
  const [username, setUsername] = useState("octocat-dev");
  const [email, setEmail] = useState("developer@example.com");
  const [displayName, setDisplayName] = useState("Octo Cat");
  const [country, setCountry] = useState("日本");
  const [password, setPassword] = useState("dummy-password");
  const [passwordConfirm, setPasswordConfirm] = useState("dummy-password");
  const [agreed, setAgreed] = useState(false);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <header className="h-16 bg-white/85 backdrop-blur border-b border-[var(--border)] flex items-center justify-between px-6 sticky top-0 z-10">
        <Link href="/01-landing" className="flex items-center gap-2 font-extrabold text-lg text-[var(--text-primary)] no-underline">
          <span className="text-xl">🐙</span>
          <span>OctoHub</span>
        </Link>
        <div className="flex items-center gap-4">
          <Link
            href="/02-signin"
            className="px-3 py-1.5 text-sm rounded-md text-[var(--text-secondary)] hover:bg-[var(--bg-muted)]"
          >
            サインイン
          </Link>
        </div>
      </header>

      <div className="max-w-[640px] mx-auto px-6 pt-8 pb-16">
        <div className="text-center mb-8">
          <div className="text-5xl leading-none">🐙</div>
          <div className="text-2xl font-semibold mt-3 mb-1">OctoHubへようこそ</div>
          <div className="text-[#57606a] text-sm">数百万の開発者と一緒に、コードで世界を変えよう</div>
        </div>

        <div className="bg-white border border-[#d0d7de] rounded-lg shadow-sm">
          <div className="px-5 py-4 border-b border-[#d0d7de]">
            <div className="font-semibold text-base">アカウントを作成</div>
          </div>
          <div className="p-5">
            <form onSubmit={handleSubmit}>
              <div className="mb-4">
                <label className="block text-sm font-medium mb-1.5">
                  ユーザー名 <span className="text-[var(--danger)]">*</span>
                </label>
                <input
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="例: octocat-dev"
                  className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm bg-white focus:outline-none focus:border-[#0969da] focus:ring-2 focus:ring-[#0969da]/20"
                />
                <div className="text-xs text-[#57606a] mt-1">英数字とハイフンが使用できます。3〜39文字。</div>
              </div>

              <div className="mb-4">
                <label className="block text-sm font-medium mb-1.5">
                  メールアドレス <span className="text-[var(--danger)]">*</span>
                </label>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="you@example.com"
                  className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm bg-white focus:outline-none focus:border-[#0969da] focus:ring-2 focus:ring-[#0969da]/20"
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="mb-4">
                  <label className="block text-sm font-medium mb-1.5">表示名</label>
                  <input
                    type="text"
                    value={displayName}
                    onChange={(e) => setDisplayName(e.target.value)}
                    placeholder="表示用の名前"
                    className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm bg-white focus:outline-none focus:border-[#0969da] focus:ring-2 focus:ring-[#0969da]/20"
                  />
                </div>
                <div className="mb-4">
                  <label className="block text-sm font-medium mb-1.5">国 / 地域</label>
                  <select
                    value={country}
                    onChange={(e) => setCountry(e.target.value)}
                    className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm bg-white focus:outline-none focus:border-[#0969da] focus:ring-2 focus:ring-[#0969da]/20"
                  >
                    <option>日本</option>
                    <option>United States</option>
                    <option>United Kingdom</option>
                    <option>Germany</option>
                  </select>
                </div>
              </div>

              <div className="mb-4">
                <label className="block text-sm font-medium mb-1.5">
                  パスワード <span className="text-[var(--danger)]">*</span>
                </label>
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm bg-white focus:outline-none focus:border-[#0969da] focus:ring-2 focus:ring-[#0969da]/20"
                />
                <div className="h-1 bg-[#eaeef2] rounded-sm mt-1.5 overflow-hidden">
                  <div className="w-3/5 h-full bg-gradient-to-r from-[#1f883d] to-[#bf8700]" />
                </div>
                <div className="text-xs text-[#57606a] mt-1">15文字以上、または8文字以上で数字と小文字を含む</div>
              </div>

              <div className="mb-4">
                <label className="block text-sm font-medium mb-1.5">
                  パスワード（確認）<span className="text-[var(--danger)]">*</span>
                </label>
                <input
                  type="password"
                  value={passwordConfirm}
                  onChange={(e) => setPasswordConfirm(e.target.value)}
                  className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm bg-white focus:outline-none focus:border-[#0969da] focus:ring-2 focus:ring-[#0969da]/20"
                />
              </div>

              <div className="flex items-start gap-2 my-5 text-xs text-[#57606a]">
                <input
                  type="checkbox"
                  id="terms"
                  checked={agreed}
                  onChange={(e) => setAgreed(e.target.checked)}
                  className="mt-0.5"
                />
                <label htmlFor="terms">
                  <Link href="/03-signup" className="text-[#0969da] no-underline">利用規約</Link>
                  および
                  <Link href="/03-signup" className="text-[#0969da] no-underline">プライバシーポリシー</Link>
                  に同意します。製品アップデートや開発者向けニュースの受信に同意します。
                </label>
              </div>

              <button
                type="submit"
                className="w-full inline-flex items-center justify-center gap-1.5 px-4 py-2.5 rounded-md bg-[#1f883d] hover:bg-[#1a7f37] text-white font-medium text-sm"
              >
                アカウントを作成 →
              </button>
            </form>

            <div className="flex items-center gap-3 my-6 text-[#57606a] text-xs before:content-[''] before:flex-1 before:h-px before:bg-[#d0d7de] after:content-[''] after:flex-1 after:h-px after:bg-[#d0d7de]">
              または
            </div>

            <Link
              href="/04-dashboard"
              className="w-full inline-flex items-center justify-center gap-1.5 px-4 py-2 rounded-md border border-[#d0d7de] bg-white hover:bg-[#f6f8fa] text-sm font-medium text-[var(--text-primary)] mb-2"
            >
              <span>🐙</span> GitHubアカウントで続行
            </Link>
            <Link
              href="/04-dashboard"
              className="w-full inline-flex items-center justify-center gap-1.5 px-4 py-2 rounded-md border border-[#d0d7de] bg-white hover:bg-[#f6f8fa] text-sm font-medium text-[var(--text-primary)]"
            >
              <span>🔑</span> SSOで続行
            </Link>
          </div>
          <div className="px-5 py-4 border-t border-[#d0d7de] bg-[#f6f8fa] rounded-b-lg">
            <div className="text-center text-sm text-[#57606a]">
              すでにアカウントをお持ちですか？{" "}
              <Link href="/02-signin" className="text-[#0969da] font-medium no-underline">サインイン</Link>
            </div>
          </div>
        </div>

        <div className="mt-8 p-4 bg-[#ddf4ff] border border-[#54aeff66] rounded-md text-xs text-[#0550ae]">
          <strong>無料アカウントで利用できる機能:</strong>
          <ul className="mt-2 pl-5 list-disc space-y-1">
            <li>無制限のパブリック・プライベートリポジトリ</li>
            <li>2,000分/月の Actions 実行時間</li>
            <li>500MBのパッケージストレージ</li>
            <li>コミュニティサポート</li>
          </ul>
        </div>
      </div>
    </div>
  );
}
