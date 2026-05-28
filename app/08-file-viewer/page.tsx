"use client";

import Link from "next/link";
import { useState } from "react";

type CodeLine = { n: number; tokens: { t: string; c?: string }[] };

const codeLines: CodeLine[] = [
  { n: 1, tokens: [{ t: "import", c: "kw" }, { t: " React " }, { t: "from", c: "kw" }, { t: " " }, { t: "'react'", c: "str" }, { t: ";" }] },
  { n: 2, tokens: [{ t: "import", c: "kw" }, { t: " styles " }, { t: "from", c: "kw" }, { t: " " }, { t: "'./Button.module.css'", c: "str" }, { t: ";" }] },
  { n: 3, tokens: [{ t: "" }] },
  { n: 4, tokens: [{ t: "// ボタンコンポーネントのProps定義", c: "com" }] },
  { n: 5, tokens: [{ t: "export interface", c: "kw" }, { t: " " }, { t: "ButtonProps", c: "fn" }, { t: " {" }] },
  { n: 6, tokens: [{ t: "  label: " }, { t: "string", c: "kw" }, { t: ";" }] },
  { n: 7, tokens: [{ t: "  variant?: " }, { t: "'primary'", c: "str" }, { t: " | " }, { t: "'secondary'", c: "str" }, { t: ";" }] },
  { n: 8, tokens: [{ t: "  disabled?: " }, { t: "boolean", c: "kw" }, { t: ";" }] },
  { n: 9, tokens: [{ t: "  onClick?: () => " }, { t: "void", c: "kw" }, { t: ";" }] },
  { n: 10, tokens: [{ t: "}" }] },
  { n: 11, tokens: [{ t: "" }] },
  { n: 12, tokens: [{ t: "export const", c: "kw" }, { t: " " }, { t: "Button", c: "fn" }, { t: ": React.FC<ButtonProps> = ({" }] },
  { n: 13, tokens: [{ t: "  label," }] },
  { n: 14, tokens: [{ t: "  variant = " }, { t: "'primary'", c: "str" }, { t: "," }] },
  { n: 15, tokens: [{ t: "  disabled = " }, { t: "false", c: "kw" }, { t: "," }] },
  { n: 16, tokens: [{ t: "  onClick," }] },
  { n: 17, tokens: [{ t: "}) => {" }] },
  { n: 18, tokens: [{ t: "  " }, { t: "const", c: "kw" }, { t: " className = " }, { t: "`${styles.btn} ${styles[variant]}`", c: "str" }, { t: ";" }] },
  { n: 19, tokens: [{ t: "" }] },
  { n: 20, tokens: [{ t: "  " }, { t: "return", c: "kw" }, { t: " (" }] },
  { n: 21, tokens: [{ t: "    <" }, { t: "button", c: "fn" }] },
  { n: 22, tokens: [{ t: "      className={className}" }] },
  { n: 23, tokens: [{ t: "      disabled={disabled}" }] },
  { n: 24, tokens: [{ t: "      onClick={onClick}" }] },
  { n: 25, tokens: [{ t: "      aria-label={label}" }] },
  { n: 26, tokens: [{ t: "    >" }] },
  { n: 27, tokens: [{ t: "      {label}" }] },
  { n: 28, tokens: [{ t: "    </" }, { t: "button", c: "fn" }, { t: ">" }] },
  { n: 29, tokens: [{ t: "  );" }] },
  { n: 30, tokens: [{ t: "};" }] },
  { n: 31, tokens: [{ t: "" }] },
  { n: 32, tokens: [{ t: "export default", c: "kw" }, { t: " Button;" }] },
];

function tokenClass(c?: string) {
  switch (c) {
    case "kw": return "text-[#cf222e]";
    case "str": return "text-[#0a3069]";
    case "fn": return "text-[#8250df]";
    case "com": return "text-[#6e7781] italic";
    case "num": return "text-[#0550ae]";
    default: return "text-[#24292f]";
  }
}

export default function FileViewerPage() {
  const [view, setView] = useState<"Code" | "Blame" | "Raw">("Code");

  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <header className="h-16 bg-white/85 backdrop-blur border-b border-[var(--border)] flex items-center justify-between px-6 sticky top-0 z-10">
        <div className="flex items-center gap-2 text-lg font-extrabold">
          <span>🐙</span>
          <span className="bg-clip-text text-transparent bg-[linear-gradient(135deg,#6366f1_0%,#8b5cf6_50%,#ec4899_100%)]">OpenHub</span>
        </div>
        <div className="flex items-center gap-3">
          <Link href="/09-commit-history" className="text-sm px-3 py-1.5 rounded-md hover:bg-[var(--bg-muted)] text-[var(--text-secondary)]">📜 履歴</Link>
          <Link href="/07-repo-detail" className="text-sm px-3 py-1.5 rounded-md hover:bg-[var(--bg-muted)] text-[var(--text-secondary)]">📁 リポジトリ</Link>
          <span className="w-6 h-6 rounded-full bg-[linear-gradient(135deg,#6e40c9,#218bff)] text-white text-[11px] font-semibold inline-flex items-center justify-center">YT</span>
        </div>
      </header>

      <nav className="px-6 py-4 bg-white border-b border-[#e1e4e8] text-sm">
        <Link href="/07-repo-detail" className="text-[#0969da] hover:underline">yamada-taro</Link>
        <span className="mx-1.5 text-[#57606a]">/</span>
        <Link href="/07-repo-detail" className="text-[#0969da] hover:underline"><strong>awesome-app</strong></Link>
        <span className="mx-1.5 text-[#57606a]">/</span>
        <Link href="/07-repo-detail" className="text-[#0969da] hover:underline">src</Link>
        <span className="mx-1.5 text-[#57606a]">/</span>
        <Link href="/07-repo-detail" className="text-[#0969da] hover:underline">components</Link>
        <span className="mx-1.5 text-[#57606a]">/</span>
        <span>Button.tsx</span>
        <span className="ml-2 inline-block px-2 py-0.5 rounded-full text-xs bg-[var(--info-light)] text-[var(--info)]">main</span>
      </nav>

      <div className="mx-6 mt-4 bg-[#f6f8fa] border border-[#d0d7de] border-b-0 px-4 py-2.5 flex items-center gap-3 text-sm">
        <span className="w-6 h-6 rounded-full bg-[linear-gradient(135deg,#6e40c9,#218bff)] text-white text-[11px] font-semibold inline-flex items-center justify-center">YT</span>
        <strong>yamada-taro</strong>
        <Link href="/09-commit-history" className="text-[#0969da] no-underline hover:underline">feat: ボタンコンポーネントにdisabled状態を追加</Link>
        <span className="ml-auto text-[var(--text-muted)]">
          <span className="font-mono">a3f7c1b</span> · 2時間前 · <Link href="/09-commit-history" className="text-[#0969da]">142 commits</Link>
        </span>
      </div>

      <div className="mx-6 mt-0 bg-white border border-[#d0d7de] rounded-t-md px-3 py-2 flex items-center justify-between">
        <div className="flex items-center gap-4 text-[13px] text-[#57606a]">
          <span>📄 <strong>Button.tsx</strong></span>
          <span>48 行</span>
          <span>1.2 KB</span>
          <span>TypeScript</span>
        </div>
        <div className="flex gap-1.5">
          <div className="inline-flex border border-[#d0d7de] rounded-md overflow-hidden">
            {(["Code", "Blame", "Raw"] as const).map((v, i) => (
              <button
                key={v}
                onClick={() => setView(v)}
                className={`px-3 py-1 text-[13px] ${view === v ? "bg-[#0969da] text-white" : "bg-white text-[#24292f]"} ${i < 2 ? "border-r border-[#d0d7de]" : ""}`}
              >
                {v}
              </button>
            ))}
          </div>
          <button className="px-2 py-1 text-sm rounded-md hover:bg-[var(--bg-muted)]" title="コピー">📋</button>
          <Link href="/09-commit-history" className="px-2 py-1 text-sm rounded-md hover:bg-[var(--bg-muted)]" title="履歴">🕐</Link>
          <button className="px-3 py-1 text-sm rounded-md bg-[var(--primary)] text-white hover:bg-[var(--primary-hover)]">✏️ 編集</button>
        </div>
      </div>

      <div className="mx-6 mb-4 border border-[#d0d7de] border-t-0 bg-white flex min-h-[600px]">
        <div className="basis-3/5 border-r border-[#e1e4e8] overflow-auto">
          <table className="w-full border-collapse font-mono text-[13px]">
            <tbody>
              {codeLines.map((line) => (
                <tr key={line.n}>
                  <td className="text-right text-[#8c959f] select-none w-12 border-r border-[#eaeef2] bg-[#f6f8fa] px-3 align-top leading-5">{line.n}</td>
                  <td className="px-3 whitespace-pre align-top leading-5">
                    {line.tokens.map((tok, idx) => (
                      <span key={idx} className={tokenClass(tok.c)}>{tok.t}</span>
                    ))}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className="flex-1 p-5 bg-[#fafbfc] overflow-auto">
          <h3 className="text-[13px] text-[#57606a] uppercase tracking-wider mb-2">📋 ファイル情報</h3>
          <dl className="text-[13px] text-[#57606a] leading-relaxed">
            <dt className="font-semibold text-[#24292f] mt-2.5">パス</dt>
            <dd className="font-mono mt-0.5">src/components/Button.tsx</dd>
            <dt className="font-semibold text-[#24292f] mt-2.5">サイズ</dt>
            <dd className="mt-0.5">1,247 bytes</dd>
            <dt className="font-semibold text-[#24292f] mt-2.5">行数</dt>
            <dd className="mt-0.5">48 行</dd>
            <dt className="font-semibold text-[#24292f] mt-2.5">言語</dt>
            <dd className="mt-0.5">TypeScript (React)</dd>
            <dt className="font-semibold text-[#24292f] mt-2.5">最終更新</dt>
            <dd className="mt-0.5">2024-01-15 14:32 (2時間前)</dd>
            <dt className="font-semibold text-[#24292f] mt-2.5">コミットハッシュ</dt>
            <dd className="font-mono mt-0.5">a3f7c1b9</dd>
          </dl>

          <h3 className="mt-6 text-[13px] text-[#57606a] uppercase tracking-wider mb-2">👥 コントリビューター</h3>
          <div className="flex gap-2 flex-wrap">
            <span className="w-6 h-6 rounded-full bg-[linear-gradient(135deg,#6e40c9,#218bff)] text-white text-[11px] font-semibold inline-flex items-center justify-center" title="yamada-taro">YT</span>
            <span className="w-6 h-6 rounded-full bg-[linear-gradient(135deg,#e85d75,#f59e0b)] text-white text-[11px] font-semibold inline-flex items-center justify-center" title="suzuki-hanako">SH</span>
            <span className="w-6 h-6 rounded-full bg-[linear-gradient(135deg,#10b981,#0ea5e9)] text-white text-[11px] font-semibold inline-flex items-center justify-center" title="tanaka-jiro">TJ</span>
          </div>

          <h3 className="mt-6 text-[13px] text-[#57606a] uppercase tracking-wider mb-2">🔗 アクション</h3>
          <div className="flex flex-col gap-2">
            <Link href="/09-commit-history" className="text-sm px-3 py-1.5 rounded-md border border-[var(--border)] text-center hover:bg-[var(--bg-muted)]">📜 このファイルの履歴</Link>
            <button className="text-sm px-3 py-1.5 rounded-md border border-[var(--border)] hover:bg-[var(--bg-muted)]">👁 Blame表示</button>
            <button className="text-sm px-3 py-1.5 rounded-md border border-[var(--border)] hover:bg-[var(--bg-muted)]">📋 パスをコピー</button>
            <button className="text-sm px-3 py-1.5 rounded-md border border-[var(--border)] hover:bg-[var(--bg-muted)]">⬇️ Raw ダウンロード</button>
          </div>
        </div>
      </div>

      <div className="bg-[#24292f] text-[#c9d1d9] px-6 py-2 flex justify-between text-xs font-mono">
        <div className="flex gap-5">
          <span>UTF-8</span>
          <span>LF</span>
          <span>TypeScript</span>
          <span>32 行 / 1,247 文字</span>
        </div>
        <div className="flex gap-5">
          <span className="text-[#3fb950]">● 保存済み</span>
          <span>最終更新: 2024-01-15 14:32</span>
          <span>main</span>
        </div>
      </div>
    </div>
  );
}
