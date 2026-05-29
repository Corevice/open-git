"use client";

import Link from "next/link";
import { useState, FormEvent } from "react";

export default function SignupPage() {
  const [username, setUsername] = useState("octocat-dev");
  const [email, setEmail] = useState("developer@example.com");
  const [displayName, setDisplayName] = useState("Octo Cat");
  const [country, setCountry] = useState("日本");
  const [password, setPassword] = useState("dummy-password");
  const [passwordConfirm, setPasswordConfirm] = useState("dummy-password");
  const [agreed, setAgreed] = useState(false);

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  const inputClass =
    "w-full px-3 py-2 border border-[color:var(--border)] rounded-md text-sm bg-white focus:outline-none focus:border-[color:var(--primary)] focus:shadow-[0_0_0_3px_rgba(99,102,241,0.2)]";
  const labelClass = "block text-sm font-medium mb-1.5 text-[color:var(--text-primary)]";
  const hintClass = "text-xs text-[color:var(--text-muted)] mt-1";
  const btnBlock =
    "w-full inline-flex items-center justify-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium transition";

  return (
    <div className="min-h-screen bg-[color:var(--bg-base)]">
      <header className="h-16 bg-white/85 backdrop-blur border-b border-[color:var(--border)] flex items-center justify-between px-6 sticky top-0 z-10">
        <Link href="/01-landing" className="flex items-center gap-2 font-extrabold text-lg text-[color:var(--text-primary)] no-underline">
          <span className="text-xl">🐙</span>
          <span>OctoHub</span>
        </Link>
        <div className="flex items-center gap-4">
          <Link href="/02-signin" className="px-3 py-1.5 text-sm text-[color:var(--text-secondary)] hover:text-[color:var(--primary)]">
            サインイン
          </Link>
        </div>
      </header>

      <div className="max-w-[640px] mx-auto px-6 pt-8 pb-16">
        <div className="text-center mb-8">
          <div className="text-5xl leading-none">🐙</div>
          <div className="text-2xl font-semibold mt-3 mb-1">OctoHubへようこそ</div>
          <div className="text-[color:var(--text-secondary)] text-sm">
            数百万の開発者と一緒に、コードで世界を変えよう
          </div>
        </div>

        <div className="bg-white border border-[color:var(--border)] rounded-lg shadow-sm overflow-hidden">
          <div className="px-6 py-4 border-b border-[color:var(--border)]">
            <div className="text-base font-semibold">アカウントを作成</div>
          </div>
          <div className="px-6 py-5">
            <form onSubmit={handleSubmit}>
              <div className="mb-4">
                <label className={labelClass}>
                  ユーザー名 <span className="text-[color:var(--danger)]">*</span>
                </label>
                <input
                  type="text"
                  className={inputClass}
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="例: octocat-dev"
                />
                <div className={hintClass}>英数字とハイフンが使用できます。3〜39文字。</div>
              </div>

              <div className="mb-4">
                <label className={labelClass}>
                  メールアドレス <span className="text-[color:var(--danger)]">*</span>
                </label>
                <input
                  type="email"
                  className={inputClass}
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="you@example.com"
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="mb-4">
                  <label className={labelClass}>表示名</label>
                  <input
                    type="text"
                    className={inputClass}
                    value={displayName}
                    onChange={(e) => setDisplayName(e.target.value)}
                    placeholder="表示用の名前"
                  />
                </div>
                <div className="mb-4">
                  <label className={labelClass}>国 / 地域</label>
                  <select
                    className={inputClass}
                    value={country}
                    onChange={(e) => setCountry(e.target.value)}
                  >
                    <option>日本</option>
                    <option>United States</option>
                    <option>United Kingdom</option>
                    <option>Germany</option>
                  </select>
                </div>
              </div>

              <div className="mb-4">
                <label className={labelClass}>
                  パスワード <span className="text-[color:var(--danger)]">*</span>
                </label>
                <input
                  type="password"
                  className={inputClass}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                />
                <div className="h-1 bg-[color:var(--bg-muted)] rounded mt-1.5 overflow-hidden">
                  <div className="w-3/5 h-full bg-gradient-to-r from-[color:var(--success)] to-[color:var(--warning)]" />
                </div>
                <div className={hintClass}>15文字以上、または8文字以上で数字と小文字を含む</div>
              </div>

              <div className="mb-4">
                <label className={labelClass}>
                  パスワード（確認）<span className="text-[color:var(--danger)]">*</span>
                </label>
                <input
                  type="password"
                  className={inputClass}
                  value={passwordConfirm}
                  onChange={(e) => setPasswordConfirm(e.target.value)}
                />
              </div>

              <div className="flex items-start gap-2 my-5 text-xs text-[color:var(--text-secondary)]">
                <input
                  type="checkbox"
                  id="terms"
                  className="mt-1"
                  checked={agreed}
                  onChange={(e) => setAgreed(e.target.checked)}
                />
                <label htmlFor="terms">
                  <Link href="/03-signup" className="text-[color:var(--primary)] no-underline">
                    利用規約
                  </Link>
                  および
                  <Link href="/03-signup" className="text-[color:var(--primary)] no-underline">
                    プライバシーポリシー
                  </Link>
                  に同意します。製品アップデートや開発者向けニュースの受信に同意します。
                </label>
              </div>

              <Link
                href="/04-dashboard"
                className={`${btnBlock} bg-[color:var(--primary)] text-white hover:bg-[color:var(--primary-hover)] py-2.5 text-base no-underline`}
              >
                アカウントを作成 →
              </Link>
            </form>

            <div className="flex items-center gap-3 my-6 text-xs text-[color:var(--text-muted)]">
              <div className="flex-1 h-px bg-[color:var(--border)]" />
              <span>または</span>
              <div className="flex-1 h-px bg-[color:var(--border)]" />
            </div>

            <Link
              href="/04-dashboard"
              className={`${btnBlock} border border-[color:var(--border)] text-[color:var(--text-primary)] hover:bg-[color:var(--bg-muted)] mb-2 no-underline`}
            >
              <span>🐙</span> GitHubアカウントで続行
            </Link>
            <Link
              href="/04-dashboard"
              className={`${btnBlock} border border-[color:var(--border)] text-[color:var(--text-primary)] hover:bg-[color:var(--bg-muted)] no-underline`}
            >
              <span>🔑</span> SSOで続行
            </Link>
          </div>
          <div className="px-6 py-4 border-t border-[color:var(--border)] bg-[color:var(--bg-muted)]">
            <div className="text-center text-sm text-[color:var(--text-secondary)]">
              すでにアカウントをお持ちですか？{" "}
              <Link href="/02-signin" className="text-[color:var(--primary)] font-medium no-underline">
                サインイン
              </Link>
            </div>
          </div>
        </div>

        <div className="mt-8 p-4 bg-[color:var(--info-light)] border border-[color:var(--info)]/40 rounded-md text-xs text-[color:var(--info)]">
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
