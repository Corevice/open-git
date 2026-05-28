"use client";

import Link from "next/link";
import { useState, FormEvent } from "react";

export default function SignInPage() {
  const [login, setLogin] = useState("");
  const [password, setPassword] = useState("");

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    // TODO: wire to API
  };

  return (
    <div className="min-h-screen bg-[#0d1117] text-[#c9d1d9] font-sans">
      <div className="max-w-[340px] mx-auto px-5 py-10">
        <div className="text-center mb-6">
          <span className="text-5xl text-[#c9d1d9]">⌥</span>
        </div>

        <h1 className="text-center text-2xl font-light mb-6 text-[#c9d1d9]">
          OpenHub にサインイン
        </h1>

        <div className="bg-[#161b22] border border-[#30363d] rounded-md p-4 mb-4">
          <form onSubmit={handleSubmit}>
            <div className="mb-4">
              <label
                className="block text-sm font-semibold mb-1.5 text-[#c9d1d9]"
                htmlFor="login"
              >
                ユーザー名またはメールアドレス
              </label>
              <input
                type="text"
                id="login"
                name="login"
                value={login}
                onChange={(e) => setLogin(e.target.value)}
                className="w-full box-border px-3 py-2 bg-[#0d1117] border border-[#30363d] rounded-md text-[#c9d1d9] text-sm leading-5"
              />
            </div>

            <div className="mb-4">
              <div className="flex justify-between items-center mb-1.5">
                <label
                  className="block text-sm font-semibold text-[#c9d1d9]"
                  htmlFor="password"
                >
                  パスワード
                </label>
                <Link
                  href="/02-signin"
                  className="text-xs text-[#58a6ff] no-underline hover:underline"
                >
                  パスワードをお忘れですか?
                </Link>
              </div>
              <input
                type="password"
                id="password"
                name="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full box-border px-3 py-2 bg-[#0d1117] border border-[#30363d] rounded-md text-[#c9d1d9] text-sm leading-5"
              />
            </div>

            <button
              type="submit"
              className="w-full bg-[#238636] hover:bg-[#2ea043] text-white border border-white/10 px-4 py-2 rounded-md text-sm font-semibold cursor-pointer text-center block box-border"
            >
              サインイン
            </button>
          </form>

          <div className="flex items-center my-4 text-xs text-[#8b949e] before:content-[''] before:flex-1 before:h-px before:bg-[#30363d] after:content-[''] after:flex-1 after:h-px after:bg-[#30363d]">
            <span className="px-3">または</span>
          </div>

          <Link
            href="/04-dashboard"
            className="flex items-center justify-center gap-2 w-full box-border px-4 py-2 bg-[#21262d] hover:bg-[#30363d] text-[#c9d1d9] border border-[#30363d] rounded-md text-sm font-medium no-underline mb-2"
          >
            <span className="text-base">🔑</span>
            <span>パスキーでサインイン</span>
          </Link>

          <Link
            href="/04-dashboard"
            className="flex items-center justify-center gap-2 w-full box-border px-4 py-2 bg-[#21262d] hover:bg-[#30363d] text-[#c9d1d9] border border-[#30363d] rounded-md text-sm font-medium no-underline mb-2"
          >
            <span className="text-base">🔐</span>
            <span>SSOでサインイン</span>
          </Link>

          <Link
            href="/04-dashboard"
            className="flex items-center justify-center gap-2 w-full box-border px-4 py-2 bg-[#21262d] hover:bg-[#30363d] text-[#c9d1d9] border border-[#30363d] rounded-md text-sm font-medium no-underline mb-2"
          >
            <span className="text-base">🐙</span>
            <span>OAuth Apps でサインイン</span>
          </Link>
        </div>

        <div className="text-center p-4 bg-[#161b22] border border-[#30363d] rounded-md text-sm text-[#c9d1d9]">
          アカウントをお持ちでない方
          <Link
            href="/03-signup"
            className="text-[#58a6ff] no-underline ml-1 hover:underline"
          >
            新規登録
          </Link>
        </div>

        <div className="mt-12 text-center text-xs text-[#8b949e]">
          <Link href="/02-signin" className="text-[#58a6ff] no-underline mx-2">
            利用規約
          </Link>
          <Link href="/02-signin" className="text-[#58a6ff] no-underline mx-2">
            プライバシー
          </Link>
          <Link href="/02-signin" className="text-[#58a6ff] no-underline mx-2">
            セキュリティ
          </Link>
          <Link href="/02-signin" className="text-[#58a6ff] no-underline mx-2">
            ヘルプ
          </Link>
        </div>
      </div>
    </div>
  );
}
