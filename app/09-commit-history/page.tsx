"use client";

import Link from "next/link";
import { useState } from "react";

type Commit = {
  sha: string;
  message: string;
  author: string;
  initials: string;
  avatarClass: string;
  time: string;
  add: number;
  del: number;
  files: number;
  selected?: boolean;
};

const commitsDay1: Commit[] = [
  {
    sha: "a3f8c91",
    message: "feat: ユーザー認証フローにOAuth2サポートを追加",
    author: "octocat",
    initials: "OC",
    avatarClass: "bg-gradient-to-br from-[#54aeff] to-[#0969da]",
    time: "2 hours ago",
    add: 142,
    del: 38,
    files: 5,
    selected: true,
  },
  {
    sha: "b7e2d10",
    message: "fix: ログインフォームのバリデーションエラーを修正",
    author: "monalisa",
    initials: "MN",
    avatarClass: "bg-gradient-to-br from-[#ffa28b] to-[#cf222e]",
    time: "5 hours ago",
    add: 24,
    del: 12,
    files: 2,
  },
  {
    sha: "c9d4e22",
    message: "docs: READMEにインストール手順とAPIリファレンスを追加",
    author: "hubot",
    initials: "HB",
    avatarClass: "bg-gradient-to-br from-[#85e89d] to-[#1a7f37]",
    time: "8 hours ago",
    add: 89,
    del: 3,
    files: 1,
  },
];

const commitsDay2: Commit[] = [
  {
    sha: "d1a5f33",
    message: "refactor: APIクライアントの共通エラーハンドリングを抽出",
    author: "octocat",
    initials: "OC",
    avatarClass: "bg-gradient-to-br from-[#54aeff] to-[#0969da]",
    time: "1 day ago",
    add: 56,
    del: 78,
    files: 4,
  },
  {
    sha: "e8b3c44",
    message: "test: ユーザーサービスの単体テストカバレッジを95%に向上",
    author: "defunkt",
    initials: "DV",
    avatarClass: "bg-gradient-to-br from-[#d2a8ff] to-[#8250df]",
    time: "1 day ago",
    add: 212,
    del: 45,
    files: 8,
  },
];

export default function Page() {
  const [baseBranch, setBaseBranch] = useState("base: main");
  const [compareBranch, setCompareBranch] = useState("compare: feature/new-ui");

  const handleCompare = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  const renderCommit = (c: Commit) => (
    <div
      key={c.sha}
      className={`flex gap-3 items-start p-4 border-b border-[#d8dee4] last:border-b-0 ${
        c.selected ? "bg-[#ddf4ff] border-l-[3px] border-l-[#0969da] pl-[13px]" : ""
      }`}
    >
      <div className={`w-8 h-8 rounded-full flex items-center justify-center text-white font-semibold text-[13px] flex-shrink-0 ${c.avatarClass}`}>
        {c.initials}
      </div>
      <div className="flex-1 min-w-0">
        <div className="text-sm font-semibold text-[#1f2328] mb-1">{c.message}</div>
        <div className="text-xs text-[#57606a]">
          <strong>{c.author}</strong> committed {c.time} ·{" "}
          <span className="font-mono text-xs bg-[#f6f8fa] px-1.5 py-0.5 rounded text-[#0969da]">{c.sha}</span> ·{" "}
          <span className="text-[#1a7f37] font-semibold">+{c.add}</span>{" "}
          <span className="text-[#cf222e] font-semibold">−{c.del}</span> · {c.files} files
        </div>
      </div>
      <button className="text-xs px-2 py-1 border border-[#d0d7de] rounded-md bg-white hover:bg-[#f6f8fa]">
        📋 Copy SHA
      </button>
    </div>
  );

  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      {/* App bar */}
      <div className="h-16 bg-white/85 backdrop-blur border-b border-[#d0d7de] flex items-center justify-between px-6 sticky top-0 z-[100]">
        <div className="text-lg font-extrabold flex items-center gap-2">
          <span>🐙</span> OpenSource GitHub
        </div>
        <div className="flex items-center gap-4">
          <Link href="/13-pr-list" className="text-sm text-[#1f2328] hover:text-[#0969da]">
            Pull Requests
          </Link>
          <Link href="/07-repo-detail" className="text-sm text-[#1f2328] hover:text-[#0969da]">
            Issues
          </Link>
          <span className="text-xs px-2 py-1 rounded-full bg-[#ddf4ff] text-[#0969da] font-medium">@octocat</span>
        </div>
      </div>

      {/* Nav tabs */}
      <div className="flex gap-1 px-6 bg-white border-b border-[#d0d7de]">
        <Link href="/07-repo-detail" className="px-4 py-3 text-sm text-[#1f2328] border-b-2 border-transparent hover:border-[#d0d7de]">
          📄 Code
        </Link>
        <Link href="/13-pr-list" className="px-4 py-3 text-sm text-[#1f2328] border-b-2 border-transparent hover:border-[#d0d7de]">
          🔀 Pull requests
        </Link>
        <Link href="/09-commit-history" className="px-4 py-3 text-sm text-[#1f2328] border-b-2 border-[#fd8c73] font-semibold">
          📜 Commits
        </Link>
        <Link href="/07-repo-detail" className="px-4 py-3 text-sm text-[#1f2328] border-b-2 border-transparent hover:border-[#d0d7de]">
          ⚙️ Settings
        </Link>
      </div>

      {/* Detail header */}
      <div className="sticky top-16 z-10 bg-white border-b border-[#d0d7de] px-6 py-4">
        <div className="text-[13px] text-[#57606a] mb-2">
          <Link href="/07-repo-detail" className="text-[#0969da] hover:underline">octocat</Link> /{" "}
          <Link href="/07-repo-detail" className="text-[#0969da] hover:underline">awesome-project</Link> /{" "}
          <strong>Commits</strong>
        </div>
        <div className="flex items-center justify-between gap-4 flex-wrap">
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-bold m-0">コミット履歴</h1>
            <span className="text-xs px-2 py-1 rounded-full bg-[#dafbe1] text-[#1a7f37] font-medium">main</span>
            <span className="text-[13px] text-[#57606a]">128 commits</span>
          </div>
          <div className="flex gap-2">
            <Link
              href="/13-pr-list"
              className="px-3 py-1.5 rounded-md bg-[#1f883d] hover:bg-[#1a7f37] text-white text-sm font-medium"
            >
              🔀 PRを作成
            </Link>
            <Link
              href="/07-repo-detail"
              className="px-3 py-1.5 rounded-md border border-[#d0d7de] bg-white hover:bg-[#f6f8fa] text-sm font-medium text-[#1f2328]"
            >
              📄 Codeタブへ戻る
            </Link>
            <button className="px-2 py-1.5 rounded-md hover:bg-[#f6f8fa] text-sm" title="共有">
              🔗
            </button>
          </div>
        </div>

        <form
          onSubmit={handleCompare}
          className="flex items-center gap-3 mt-3 p-3 bg-[#f6f8fa] border border-[#d0d7de] rounded-md flex-wrap"
        >
          <span className="text-[13px] font-semibold">🔀 ブランチ比較:</span>
          <select
            value={baseBranch}
            onChange={(e) => setBaseBranch(e.target.value)}
            className="px-2.5 py-1.5 border border-[#d0d7de] rounded-md bg-white text-[13px]"
          >
            <option>base: main</option>
            <option>base: develop</option>
            <option>base: v1.0.0</option>
          </select>
          <span className="text-[#57606a]">←→</span>
          <select
            value={compareBranch}
            onChange={(e) => setCompareBranch(e.target.value)}
            className="px-2.5 py-1.5 border border-[#d0d7de] rounded-md bg-white text-[13px]"
          >
            <option>compare: feature/new-ui</option>
            <option>compare: feature/auth</option>
            <option>compare: hotfix/bug-123</option>
          </select>
          <button
            type="submit"
            className="px-3 py-1 text-xs rounded-md bg-[#f6f8fa] border border-[#d0d7de] hover:bg-[#eaeef2]"
          >
            比較
          </button>
          <span className="ml-auto text-xs text-[#57606a]">✓ Able to merge — 3 commits ahead</span>
        </form>
      </div>

      {/* Main grid */}
      <div className="max-w-[1280px] mx-auto px-6">
        <div className="grid grid-cols-1 lg:grid-cols-[1fr_360px] gap-6 py-6">
          <div>
            {/* Commit list */}
            <div className="bg-white border border-[#d0d7de] rounded-md">
              <div className="px-4 py-3 border-b border-[#d0d7de] font-semibold text-sm flex justify-between items-center">
                <span>📅 Oct 28, 2024</span>
                <span className="text-[#57606a] font-normal text-xs">3 commits</span>
              </div>
              {commitsDay1.map(renderCommit)}

              <div className="px-4 py-3 border-y border-[#d0d7de] font-semibold text-sm flex justify-between items-center">
                <span>📅 Oct 27, 2024</span>
                <span className="text-[#57606a] font-normal text-xs">2 commits</span>
              </div>
              {commitsDay2.map(renderCommit)}
            </div>

            {/* Diff section */}
            <div className="mt-6 bg-white border border-[#d0d7de] rounded-md">
              <div className="p-4 border-b border-[#d0d7de]">
                <h2 className="text-base font-semibold m-0 mb-2">
                  📝 Diff: <span className="font-mono">a3f8c91</span> — feat: ユーザー認証フローにOAuth2サポートを追加
                </h2>
                <div className="flex gap-4 text-[13px] text-[#57606a] flex-wrap">
                  <span><strong>5 files changed</strong></span>
                  <span className="text-[#1a7f37] font-semibold">+142 additions</span>
                  <span className="text-[#cf222e] font-semibold">−38 deletions</span>
                  <span className="text-[#57606a]">Showing 2 of 5 files</span>
                </div>
              </div>

              {/* File 1 */}
              <div>
                <div className="px-4 py-2.5 bg-[#f6f8fa] flex justify-between items-center font-mono text-[13px] border-b border-[#d0d7de]">
                  <span>
                    📄 src/auth/oauth.ts{" "}
                    <span className="text-xs px-2 py-0.5 rounded-full bg-[#dafbe1] text-[#1a7f37] font-medium ml-1">new</span>
                  </span>
                  <span>
                    <span className="text-[#1a7f37] font-semibold">+87</span>{" "}
                    <span className="text-[#cf222e] font-semibold">−0</span>
                  </span>
                </div>
                <table className="w-full border-collapse font-mono text-xs">
                  <tbody>
                    <tr className="bg-[#ddf4ff] text-[#57606a]">
                      <td className="w-[50px] text-right bg-[#f6f8fa] border-r border-[#d0d7de] px-2.5 py-1"></td>
                      <td className="px-2.5 py-1 whitespace-pre">@@ -0,0 +1,12 @@</td>
                    </tr>
                    {[
                      "import { OAuth2Client } from 'google-auth-library';",
                      "",
                      "export class OAuthProvider {",
                      "  private client: OAuth2Client;",
                      "",
                      "  constructor(clientId: string, clientSecret: string) {",
                      "    this.client = new OAuth2Client(clientId, clientSecret);",
                      "  }",
                      "",
                      "  async verifyToken(token: string): Promise<UserInfo> {",
                      "    return await this.client.verifyIdToken({ idToken: token });",
                      "  }",
                    ].map((line, i) => (
                      <tr key={i} className="bg-[#dafbe1]">
                        <td className="w-[50px] text-right bg-[#ccffd8] text-[#8c959f] border-r border-[#d0d7de] px-2.5 py-px select-none">
                          {i + 1}
                        </td>
                        <td className="px-2.5 py-px whitespace-pre">+ {line}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* File 2 */}
              <div className="border-t border-[#d0d7de]">
                <div className="px-4 py-2.5 bg-[#f6f8fa] flex justify-between items-center font-mono text-[13px] border-b border-[#d0d7de]">
                  <span>
                    📄 src/auth/login.ts{" "}
                    <span className="text-xs px-2 py-0.5 rounded-full bg-[#fef3c7] text-[#9a6700] font-medium ml-1">modified</span>
                  </span>
                  <span>
                    <span className="text-[#1a7f37] font-semibold">+32</span>{" "}
                    <span className="text-[#cf222e] font-semibold">−18</span>
                  </span>
                </div>
                <table className="w-full border-collapse font-mono text-xs">
                  <tbody>
                    <tr className="bg-[#ddf4ff] text-[#57606a]">
                      <td className="w-[50px] text-right bg-[#f6f8fa] border-r border-[#d0d7de] px-2.5 py-1"></td>
                      <td className="px-2.5 py-1 whitespace-pre">@@ -15,8 +15,12 @@ export async function login(req, res) {"}</td>
                    </tr>
                    <tr>
                      <td className="w-[50px] text-right bg-[#f6f8fa] text-[#8c959f] border-r border-[#d0d7de] px-2.5 py-px select-none">15</td>
                      <td className="px-2.5 py-px whitespace-pre">  const {"{ email, password }"} = req.body;</td>
                    </tr>
                    <tr>
                      <td className="w-[50px] text-right bg-[#f6f8fa] text-[#8c959f] border-r border-[#d0d7de] px-2.5 py-px select-none">16</td>
                      <td className="px-2.5 py-px whitespace-pre">  const user = await findUser(email);</td>
                    </tr>
                    {[
                      { n: 17, t: "if (!user) return res.status(401).send('Invalid');" },
                      { n: 18, t: "if (!checkPassword(password, user.hash)) {" },
                      { n: 19, t: "  return res.status(401).send('Invalid');" },
                      { n: 20, t: "}" },
                    ].map((l) => (
                      <tr key={`d${l.n}`} className="bg-[#ffebe9]">
                        <td className="w-[50px] text-right bg-[#ffd7d5] text-[#8c959f] border-r border-[#d0d7de] px-2.5 py-px select-none">{l.n}</td>
                        <td className="px-2.5 py-px whitespace-pre">- {l.t}</td>
                      </tr>
                    ))}
                    {[
                      { n: 17, t: "if (!user) {" },
                      { n: 18, t: "  return res.status(401).json({ error: 'INVALID_CREDENTIALS' });" },
                      { n: 19, t: "}" },
                      { n: 20, t: "const isValid = await checkPassword(password, user.hash);" },
                      { n: 21, t: "if (!isValid) {" },
                      { n: 22, t: "  return res.status(401).json({ error: 'INVALID_CREDENTIALS' });" },
                      { n: 23, t: "}" },
                    ].map((l) => (
                      <tr key={`a${l.n}`} className="bg-[#dafbe1]">
                        <td className="w-[50px] text-right bg-[#ccffd8] text-[#8c959f] border-r border-[#d0d7de] px-2.5 py-px select-none">{l.n}</td>
                        <td className="px-2.5 py-px whitespace-pre">+ {l.t}</td>
                      </tr>
                    ))}
                    <tr>
                      <td className="w-[50px] text-right bg-[#f6f8fa] text-[#8c959f] border-r border-[#d0d7de] px-2.5 py-px select-none">24</td>
                      <td className="px-2.5 py-px whitespace-pre">  const token = generateToken(user);</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>

          {/* Sidebar */}
          <aside>
            <div className="bg-white border border-[#d0d7de] rounded-md mb-4">
              <div className="px-4 py-3 border-b border-[#d0d7de] font-semibold text-sm">コミット情報</div>
              <div className="p-4 text-[13px]">
                <div className="flex justify-between py-1.5">
                  <span className="text-[#57606a]">SHA</span>
                  <span className="font-mono text-[#0969da]">a3f8c91</span>
                </div>
                <div className="flex justify-between py-1.5">
                  <span className="text-[#57606a]">Author</span>
                  <span>octocat</span>
                </div>
                <div className="flex justify-between py-1.5">
                  <span className="text-[#57606a]">Date</span>
                  <span>Oct 28, 2024</span>
                </div>
                <div className="flex justify-between py-1.5">
                  <span className="text-[#57606a]">Parent</span>
                  <span className="font-mono text-[#0969da]">b7e2d10</span>
                </div>
                <div className="flex justify-between py-1.5">
                  <span className="text-[#57606a]">Branch</span>
                  <span className="text-xs px-2 py-0.5 rounded-full bg-[#dafbe1] text-[#1a7f37] font-medium">main</span>
                </div>
              </div>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-md mb-4">
              <div className="px-4 py-3 border-b border-[#d0d7de] font-semibold text-sm">タイムライン</div>
              <div className="p-4">
                {[
                  "Branch created — feature/oauth",
                  "3 commits pushed",
                  "CI checks passed ✓",
                  "Ready to merge",
                ].map((t) => (
                  <div key={t} className="flex gap-2.5 py-2 text-[13px]">
                    <div className="w-2 h-2 rounded-full bg-[#0969da] mt-1.5 flex-shrink-0" />
                    <span>{t}</span>
                  </div>
                ))}
              </div>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-md">
              <div className="px-4 py-3 border-b border-[#d0d7de] font-semibold text-sm">関連リンク</div>
              <div className="p-4 text-[13px] flex flex-col gap-2">
                <Link href="/13-pr-list" className="text-[#0969da] hover:underline">🔀 関連するPull Request #42</Link>
                <Link href="/07-repo-detail" className="text-[#0969da] hover:underline">📁 ファイルツリーを表示</Link>
                <Link href="/09-commit-history" className="text-[#0969da] hover:underline">📜 全コミット履歴</Link>
              </div>
            </div>
          </aside>
        </div>
      </div>
    </div>
  );
}
