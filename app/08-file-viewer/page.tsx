import Link from "next/link";

const codeLines: { tokens: { text: string; cls?: string }[] }[] = [
  { tokens: [{ text: "import", cls: "kw" }, { text: " React " }, { text: "from", cls: "kw" }, { text: " " }, { text: "'react'", cls: "str" }, { text: ";" }] },
  { tokens: [{ text: "import", cls: "kw" }, { text: " styles " }, { text: "from", cls: "kw" }, { text: " " }, { text: "'./Button.module.css'", cls: "str" }, { text: ";" }] },
  { tokens: [{ text: "" }] },
  { tokens: [{ text: "// ボタンコンポーネントのProps定義", cls: "com" }] },
  { tokens: [{ text: "export interface", cls: "kw" }, { text: " " }, { text: "ButtonProps", cls: "fn" }, { text: " {" }] },
  { tokens: [{ text: "  label: " }, { text: "string", cls: "kw" }, { text: ";" }] },
  { tokens: [{ text: "  variant?: " }, { text: "'primary'", cls: "str" }, { text: " | " }, { text: "'secondary'", cls: "str" }, { text: ";" }] },
  { tokens: [{ text: "  disabled?: " }, { text: "boolean", cls: "kw" }, { text: ";" }] },
  { tokens: [{ text: "  onClick?: () => " }, { text: "void", cls: "kw" }, { text: ";" }] },
  { tokens: [{ text: "}" }] },
  { tokens: [{ text: "" }] },
  { tokens: [{ text: "export const", cls: "kw" }, { text: " " }, { text: "Button", cls: "fn" }, { text: ": React.FC<ButtonProps> = ({" }] },
  { tokens: [{ text: "  label," }] },
  { tokens: [{ text: "  variant = " }, { text: "'primary'", cls: "str" }, { text: "," }] },
  { tokens: [{ text: "  disabled = " }, { text: "false", cls: "kw" }, { text: "," }] },
  { tokens: [{ text: "  onClick," }] },
  { tokens: [{ text: "}) => {" }] },
  { tokens: [{ text: "  " }, { text: "const", cls: "kw" }, { text: " className = " }, { text: "`${styles.btn} ${styles[variant]}`", cls: "str" }, { text: ";" }] },
  { tokens: [{ text: "" }] },
  { tokens: [{ text: "  " }, { text: "return", cls: "kw" }, { text: " (" }] },
  { tokens: [{ text: "    <" }, { text: "button", cls: "fn" }] },
  { tokens: [{ text: "      className={className}" }] },
  { tokens: [{ text: "      disabled={disabled}" }] },
  { tokens: [{ text: "      onClick={onClick}" }] },
  { tokens: [{ text: "      aria-label={label}" }] },
  { tokens: [{ text: "    >" }] },
  { tokens: [{ text: "      {label}" }] },
  { tokens: [{ text: "    </" }, { text: "button", cls: "fn" }, { text: ">" }] },
  { tokens: [{ text: "  );" }] },
  { tokens: [{ text: "};" }] },
  { tokens: [{ text: "" }] },
  { tokens: [{ text: "export default", cls: "kw" }, { text: " Button;" }] },
];

function tokenClass(cls?: string) {
  switch (cls) {
    case "kw":
      return "text-[#cf222e]";
    case "str":
      return "text-[#0a3069]";
    case "fn":
      return "text-[#8250df]";
    case "com":
      return "text-[#6e7781] italic";
    case "num":
      return "text-[#0550ae]";
    default:
      return "text-[#24292f]";
  }
}

export default function FileViewerPage() {
  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      {/* App Bar */}
      <header className="h-16 sticky top-0 z-50 flex items-center justify-between px-6 border-b border-[var(--border)] bg-white/85 backdrop-blur">
        <div className="flex items-center gap-2 text-lg font-extrabold">
          <span>🐙</span>
          <span className="bg-[var(--gradient-primary)] bg-clip-text text-transparent">OpenHub</span>
        </div>
        <div className="flex items-center gap-3">
          <Link href="/09-commit-history" className="px-3 py-1.5 text-sm rounded-md hover:bg-[var(--bg-muted)] text-[var(--text-primary)]">📜 履歴</Link>
          <Link href="/07-repo-detail" className="px-3 py-1.5 text-sm rounded-md hover:bg-[var(--bg-muted)] text-[var(--text-primary)]">📁 リポジトリ</Link>
          <span className="w-6 h-6 rounded-full bg-gradient-to-br from-[#6e40c9] to-[#218bff] inline-flex items-center justify-center text-white text-[11px] font-semibold">YT</span>
        </div>
      </header>

      {/* Breadcrumb */}
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

      {/* Commit bar */}
      <div className="mx-6 mt-0 bg-[#f6f8fa] border border-[#d0d7de] border-t-0 px-4 py-2.5 flex items-center gap-3 text-sm">
        <span className="w-6 h-6 rounded-full bg-gradient-to-br from-[#6e40c9] to-[#218bff] inline-flex items-center justify-center text-white text-[11px] font-semibold">YT</span>
        <strong>yamada-taro</strong>
        <Link href="/09-commit-history" className="text-[#0969da] hover:underline">feat: ボタンコンポーネントにdisabled状態を追加</Link>
        <span className="ml-auto text-[var(--text-muted)]">
          <span className="font-mono">a3f7c1b</span> · 2時間前 · <Link href="/09-commit-history" className="text-[#0969da] hover:underline">142 commits</Link>
        </span>
      </div>

      {/* File toolbar */}
      <div className="mx-6 mt-4 bg-white border border-[#d0d7de] rounded-t-md px-3 py-2 flex items-center justify-between">
        <div className="flex items-center gap-4 text-[13px] text-[#57606a]">
          <span>📄 <strong className="text-[#24292f]">Button.tsx</strong></span>
          <span>48 行</span>
          <span>1.2 KB</span>
          <span>TypeScript</span>
        </div>
        <div className="flex gap-1.5 items-center">
          <div className="inline-flex border border-[#d0d7de] rounded-md overflow-hidden text-[13px]">
            <Link href="/08-file-viewer" className="px-3 py-1 bg-[#0969da] text-white">Code</Link>
            <Link href="/08-file-viewer" className="px-3 py-1 bg-white text-[#24292f] border-l border-[#d0d7de]">Blame</Link>
            <Link href="/08-file-viewer" className="px-3 py-1 bg-white text-[#24292f] border-l border-[#d0d7de]">Raw</Link>
          </div>
          <Link href="/08-file-viewer" title="コピー" className="px-2 py-1 text-sm rounded-md hover:bg-[var(--bg-muted)]">📋</Link>
          <Link href="/09-commit-history" title="履歴" className="px-2 py-1 text-sm rounded-md hover:bg-[var(--bg-muted)]">🕐</Link>
          <Link href="/08-file-viewer" className="px-2.5 py-1 text-sm rounded-md bg-[var(--primary)] text-white hover:bg-[var(--primary-hover)]">✏️ 編集</Link>
        </div>
      </div>

      {/* Editor wrap */}
      <div className="mx-6 mb-4 border border-[#d0d7de] border-t-0 bg-white flex min-h-[600px]">
        <div className="flex-[0_0_60%] border-r border-[#e1e4e8] overflow-auto">
          <table className="w-full border-collapse font-mono text-[13px]">
            <tbody>
              {codeLines.map((line, i) => (
                <tr key={i}>
                  <td className="px-3 py-0 text-right text-[#8c959f] select-none w-12 border-r border-[#eaeef2] bg-[#f6f8fa] leading-5 align-top">{i + 1}</td>
                  <td className="px-3 py-0 whitespace-pre text-[#24292f] leading-5 align-top">
                    {line.tokens.map((t, ti) => (
                      <span key={ti} className={tokenClass(t.cls)}>{t.text}</span>
                    ))}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="flex-1 p-5 bg-[#fafbfc] overflow-auto">
          <h3 className="text-[13px] text-[#57606a] uppercase tracking-wider mb-2 font-semibold">📋 ファイル情報</h3>
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

          <h3 className="text-[13px] text-[#57606a] uppercase tracking-wider mt-6 mb-2 font-semibold">👥 コントリビューター</h3>
          <div className="flex gap-2 flex-wrap">
            <span title="yamada-taro" className="w-6 h-6 rounded-full bg-gradient-to-br from-[#6e40c9] to-[#218bff] inline-flex items-center justify-center text-white text-[11px] font-semibold">YT</span>
            <span title="suzuki-hanako" className="w-6 h-6 rounded-full bg-gradient-to-br from-[#e85d75] to-[#f59e0b] inline-flex items-center justify-center text-white text-[11px] font-semibold">SH</span>
            <span title="tanaka-jiro" className="w-6 h-6 rounded-full bg-gradient-to-br from-[#10b981] to-[#0ea5e9] inline-flex items-center justify-center text-white text-[11px] font-semibold">TJ</span>
          </div>

          <h3 className="text-[13px] text-[#57606a] uppercase tracking-wider mt-6 mb-2 font-semibold">🔗 アクション</h3>
          <div className="flex flex-col gap-2">
            <Link href="/09-commit-history" className="px-3 py-1.5 text-sm rounded-md border border-[var(--border-strong)] text-[var(--text-primary)] hover:bg-[var(--bg-muted)] text-center">📜 このファイルの履歴</Link>
            <Link href="/08-file-viewer" className="px-3 py-1.5 text-sm rounded-md border border-[var(--border-strong)] text-[var(--text-primary)] hover:bg-[var(--bg-muted)] text-center">👁 Blame表示</Link>
            <Link href="/08-file-viewer" className="px-3 py-1.5 text-sm rounded-md border border-[var(--border-strong)] text-[var(--text-primary)] hover:bg-[var(--bg-muted)] text-center">📋 パスをコピー</Link>
            <Link href="/08-file-viewer" className="px-3 py-1.5 text-sm rounded-md border border-[var(--border-strong)] text-[var(--text-primary)] hover:bg-[var(--bg-muted)] text-center">⬇️ Raw ダウンロード</Link>
          </div>
        </div>
      </div>

      {/* Status bar */}
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
