"use client";

import Link from "next/link";
import { useState } from "react";

type Commit = {
  sha: string;
  msg: string;
  author: string;
  initials: string;
  avatarClass: string;
  time: string;
  adds: number;
  dels: number;
  files: number;
};

const day1: Commit[] = [
  { sha: "a3f8c91", msg: "feat: ユーザー認証フローにOAuth2サポートを追加", author: "octocat", initials: "OC", avatarClass: "bg-gradient-to-br from-[#54aeff] to-[#0969da]", time: "2 hours ago", adds: 142, dels: 38, files: 5 },
  { sha: "b7e2d10", msg: "fix: ログインフォームのバリデーションエラーを修正", author: "monalisa", initials: "MN", avatarClass: "bg-gradient-to-br from-[#ffa28b] to-[#cf222e]", time: "5 hours ago", adds: 24, dels: 12, files: 2 },
  { sha: "c9d4e22", msg: "docs: READMEにインストール手順とAPIリファレンスを追加", author: "hubot", initials: "HB", avatarClass: "bg-gradient-to-br from-[#85e89d] to-[#1a7f37]", time: "8 hours ago", adds: 89, dels: 3, files: 1 },
];

const day2: Commit[] = [
  { sha: "d1a5f33", msg: "refactor: APIクライアントの共通エラーハンドリングを抽出", author: "octocat", initials: "OC", avatarClass: "bg-gradient-to-br from-[#54aeff] to-[#0969da]", time: "1 day ago", adds: 56, dels: 78, files: 4 },
  { sha: "e8b3c44", msg: "test: ユーザーサービスの単体テストカバレッジを95%に向上", author: "defunkt", initials: "DV", avatarClass: "bg-gradient-to-br from-[#d2a8ff] to-[#8250df]", time: "1 day ago", adds: 212, dels: 45, files: 8 },
];

export default function Page() {
  const [base, setBase] = useState("base: main");
  const [compare, setCompare] = useState("compare: feature/new-ui");
  const [selectedSha, setSelectedSha] = useState("a3f8c91");

  const handleCompare = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  const renderCommit = (c: Commit) => {
    const isSelected = c.sha === selectedSha;
    return (
      <div
        key={c.sha}
        onClick={() => setSelectedSha(c.sha)}
        className={`flex gap-3 items-start p-4 border-b border-[#d8dee4] last:border-b-0 cursor-pointer ${isSelected ? "bg-[#ddf4ff] border-l-[3px] border-l-[#0969da] pl-[13px]" : ""}`}
      >
        <div className={`w-8 h-8 rounded-full flex items-center justify-center text-white font-semibold text-[13px] flex-shrink-0 ${c.avatarClass}`}>
          {c.initials}
        </div>
        <div className="flex-1 min-w-0">
          <div className="text-sm font-semibold text-[#1f2328] mb-1">{c.msg}</div>
          <div className="text-xs text-[#57606a]">
            <strong>{c.author}</strong> committed {c.time} ·{" "}
            <span className="font-mono text-xs bg-[#f6f8fa] px-1.5 py-0.5 rounded text-[#0969da]">{c.sha}</span> ·{" "}
            <span className="text-[#1a7f37] font-semibold">+{c.adds}</span>{" "}
            <span className="text-[#cf222e] font-semibold">−{c.dels}</span> · {c.files} files
          </div>
        </div>
        <button className="text-xs px-2 py-1 border border-[#d0d7de] rounded bg-white hover:bg-[#f6f8fa]">📋 Copy SHA</button>
      </div>
    );
  };

  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      {/* App Bar */}
      <div className="h-16 bg-white/85 backdrop-blur border-b border-[#d0d7de] flex items-center justify-between px-6 sticky top-0 z-[100]">
        <div className="text-lg font-extrabold flex items-center gap-2">
          <span>🐙</span> OpenSource GitHub
        </div>
        <div className="flex items-center gap-4">
          <Link href="/13-pr-list" className="text-sm px-3 py-1.5 rounded hover:bg-[#f6f8fa] text-[#1f2328]">Pull Requests</Link>
          <Link href="/07-repo-detail" className="text-sm px-3 py-1.5 rounded hover:bg-[#f6f8fa] text-[#1f2328]">Issues</Link>
          <span className="text-xs px-2 py-1 rounded bg-[#dbeafe] text-[#1e40af]">@octocat</span>
        </div>
      </div>

      {/* Nav Tabs */}
      <div className="flex gap-1 px-6 bg-white border-b border-[#d0d7de]">
        <Link href="/07-repo-detail" className="px-4 py-3 text-sm text-[#1f2328] border-b-2 border-transparent">📄 Code</Link>
        <Link href="/13-pr-list" className="px-4 py-3 text-sm text-[#1f2328] border-b-2 border-transparent">🔀 Pull requests</Link>
        <Link href="/09-commit-history" className="px-4 py-3 text-sm text-[#1f2328] border-b-2 border-[#fd8c73] font-semibold">📜 Commits</Link>
        <Link href="/07-repo-detail" className="px-4 py-3 text-sm text-[#1f2328] border-b-2 border-transparent">⚙️ Settings</Link>
      </div>

      {/* Detail Header */}
      <div className="sticky top-16 z-10 bg-white border-b border-[#d0d7de] px-6 py-4">
        <div className="text-[13px] text-[#57606a] mb-2">
          <Link href="/07-repo-detail" className="text-[#0969da]">octocat</Link> /{" "}
          <Link href="/07-repo-detail" className="text-[#0969da]">awesome-project</Link> / <strong>Commits</strong>
        </div>
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-bold m-0">コミット履歴</h1>
            <span className="text-xs px-2 py-0.5 rounded bg-[#d1fae5] text-[#065f46]">main</span>
            <span className="text-[13px] text-[#94a3b8]">128 commits</span>
          </div>
          <div className="flex gap-2">
            <Link href="/13-pr-list" className="text-sm px-3 py-1.5 rounded bg-[#0969da] text-white hover:bg-[#0860c7]">🔀 PRを作成</Link>
            <Link href="/07-repo-detail" className="text-sm px-3 py-1.5 rounded border border-[#d0d7de] bg-white text-[#1f2328] hover:bg-[#f6f8fa]">📄 Codeタブへ戻る</Link>
            <button className="text-sm px-2 py-1.5 rounded hover:bg-[#f6f8fa]" title="共有">🔗</button>
          </div>
        </div>

        {/* Compare Bar */}
        <form onSubmit={handleCompare} className="flex items-center gap-3 mt-3 p-3 bg-[#f6f8fa] border border-[#d0d7de] rounded-md">
          <span className="text-[13px] font-semibold">🔀 ブランチ比較:</span>
          <select value={base} onChange={(e) => setBase(e.target.value)} className="px-2.5 py-1.5 border border-[#d0d7de] rounded bg-white text-[13px]">
            <option>base: main</option>
            <option>base: develop</option>
            <option>base: v1.0.0</option>
          </select>
          <span className="text-[#94a3b8]">←→</span>
          <select value={compare} onChange={(e) => setCompare(e.target.value)} className="px-2.5 py-1.5 border border-[#d0d7de] rounded bg-white text-[13px]">
            <option>compare: feature/new-ui</option>
            <option>compare: feature/auth</option>
            <option>compare: hotfix/bug-123</option>
          </select>
          <button type="submit" className="text-xs px-3 py-1.5 rounded bg-[#6e7681] text-white hover:bg-[#57606a]">比較</button>
          <span className="ml-auto text-xs text-[#94a3b8]">✓ Able to merge — 3 commits ahead</span>
        </form>
      </div>

      {/* Main */}
      <div className="max-w-[1280px] mx-auto px-6">
        <div className="grid grid-cols-1 lg:grid-cols-[1fr_360px] gap-6 py-6">
          <div>
            {/* Commit list */}
            <div className="bg-white border border-[#d0d7de] rounded-md">
              <div className="px-4 py-3 border-b border-[#d0d7de] font-semibold text-sm flex justify-between items-center">
                <span>📅 Oct 28, 2024</span>
                <span className="text-[#94a3b8] font-normal text-xs">3 commits</span>
              </div>
              {day1.map(renderCommit)}

              <div className="px-4 py-3 border-t border-b border-[#d0d7de] font-semibold text-sm flex justify-between items-center">
                <span>📅 Oct 27, 2024</span>
                <span className="text-[#94a3b8] font-normal text-xs">2 commits</span>
              </div>
              {day2.map(renderCommit)}
            </div>

            {/* Diff section */}
            <div className="mt-6 bg-white border border-[#d0d7de] rounded-md">
              <div className="p-4 border-b border-[#d0d7de]">
                <h2 className="text-base font-semibold mb-2">📝 Diff: <span className="font-mono">a3f8c91</span> — feat: ユーザー認証フローにOAuth2サポートを追加</h2>
                <div className="flex gap-4 text-[13px] text-[#57606a]">
                  <span><strong>5 files changed</strong></span>
                  <span className="text-[#1a7f37] font-semibold">+142 additions</span>
                  <span className="text-[#cf222e] font-semibold">−38 deletions</span>
                  <span className="text-[#94a3b8]">Showing 2 of 5 files</span>
                </div>
              </div>

              {/* File 1 */}
              <div className="border-t border-[#d0d7de] first:border-t-0">
                <div className="px-4 py-2.5 bg-[#f6f8fa] flex justify-between items-center font-mono text-[13px]">
                  <span>📄 src/auth/oauth.ts <span className="text-xs px-1.5 py-0.5 rounded bg-[#d1fae5] text-[#065f46] ml-1">new</span></span>
                  <span><span className="text-[#1a7f37] font-semibold">+87</span> <span className="text-[#cf222e] font-semibold">−0</span></span>
                </div>
                <table className="w-full border-collapse font-mono text-xs">
                  <tbody>
                    <tr className="bg-[#ddf4ff] text-[#57606a]"><td className="w-[50px] text-right px-2.5 py-1 border-r border-[#d0d7de]"></td><td className="px-2.5 py-1 whitespace-pre">@@ -0,0 +1,12 @@</td></tr>
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
                        <td className="w-[50px] text-right px-2.5 py-0.5 text-[#8c959f] bg-[#ccffd8] border-r border-[#d0d7de] select-none">{i + 1}</td>
                        <td className="px-2.5 py-0.5 whitespace-pre">+ {line}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* File 2 */}
              <div className="border-t border-[#d0d7de]">
                <div className="px-4 py-2.5 bg-[#f6f8fa] flex justify-between items-center font-mono text-[13px]">
                  <span>📄 src/auth/login.ts <span className="text-xs px-1.5 py-0.5 rounded bg-[#fef3c7] text-[#92400e] ml-1">modified</span></span>
                  <span><span className="text-[#1a7f37] font-semibold">+32</span> <span className="text-[#cf222e] font-semibold">−18</span></span>
                </div>
                <table className="w-full border-collapse font-mono text-xs">
                  <tbody>
                    <tr className="bg-[#ddf4ff] text-[#57606a]"><td className="w-[50px] text-right px-2.5 py-1 border-r border-[#d0d7de]"></td><td className="px-2.5 py-1 whitespace-pre">@@ -15,8 +15,12 @@ export async function login(req, res) {"}</td></tr>
                    <tr><td className="w-[50px] text-right px-2.5 py-0.5 text-[#8c959f] bg-[#f6f8fa] border-r border-[#d0d7de] select-none">15</td><td className="px-2.5 py-0.5 whitespace-pre">  const &#123; email, password &#125; = req.body;</td></tr>
                    <tr><td className="w-[50px] text-right px-2.5 py-0.5 text-[#8c959f] bg-[#f6f8fa] border-r border-[#d0d7de] select-none">16</td><td className="px-2.5 py-0.5 whitespace-pre">  const user = await findUser(email);</td></tr>
                    {[
                      ["17", "- if (!user) return res.status(401).send('Invalid');"],
                      ["18", "- if (!checkPassword(password, user.hash)) {"],
                      ["19", "-   return res.status(401).send('Invalid');"],
                      ["20", "- }"],
                    ].map(([n, t]) => (
                      <tr key={n} className="bg-[#ffebe9]">
                        <td className="w-[50px] text-right px-2.5 py-0.5 text-[#8c959f] bg-[#ffd7d5] border-r border-[#d0d7de] select-none">{n}</td>
                        <td className="px-2.5 py-0.5 whitespace-pre">{t}</td>
                      </tr>
                    ))}
                    {[
                      ["17", "+ if (!user) {"],
                      ["18", "+   return res.status(401).json({ error: 'INVALID_CREDENTIALS' });"],
                      ["19", "+ }"],
                      ["20", "+ const isValid = await checkPassword(password, user.hash);"],
                      ["21", "+ if (!isValid) {"],
                      ["22", "+   return res.status(401).json({ error: 'INVALID_CREDENTIALS' });"],
                      ["23", "+ }"],
                    ].map(([n, t]) => (
                      <tr key={n} className="bg-[#dafbe1]">
                        <td className="w-[50px] text-right px-2.5 py-0.5 text-[#8c959f] bg-[#ccffd8] border-r border-[#d0d7de] select-none">{n}</td>
                        <td className="px-2.5 py-0.5 whitespace-pre">{t}</td>
                      </tr>
                    ))}
                    <tr><td className="w-[50px] text-right px-2.5 py-0.5 text-[#8c959f] bg-[#f6f8fa] border-r border-[#d0d7de] select-none">24</td><td className="px-2.5 py-0.5 whitespace-pre">  const token = generateToken(user);</td></tr>
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
                <div className="flex justify-between py-1.5"><span className="text-[#57606a]">SHA</span><span className="font-mono text-[#0969da]">a3f8c91</span></div>
                <div className="flex justify-between py-1.5"><span className="text-[#57606a]">Author</span><span>octocat</span></div>
                <div className="flex justify-between py-1.5"><span className="text-[#57606a]">Branch</span><span>main</span></div>
                <div className="flex justify-between py-1.5"><span className="text-[#57606a]">Parent</span><span className="font-mono text-[#0969da]">b7e2d10</span></div>
              </div>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-md mb-4">
              <div className="px-4 py-3 border-b border-[#d0d7de] font-semibold text-sm">関連アクティビティ</div>
              <div className="p-4">
                <div className="flex gap-2.5 py-2 text-[13px]">
                  <div className="w-2 h-2 rounded-full bg-[#0969da] mt-1.5 flex-shrink-0"></div>
                  <div>CI/CD passed · 2h ago</div>
                </div>
                <div className="flex gap-2.5 py-2 text-[13px]">
                  <div className="w-2 h-2 rounded-full bg-[#0969da] mt-1.5 flex-shrink-0"></div>
                  <div>Deployed to staging · 1h ago</div>
                </div>
                <div className="flex gap-2.5 py-2 text-[13px]">
                  <div className="w-2 h-2 rounded-full bg-[#0969da] mt-1.5 flex-shrink-0"></div>
                  <div>Reviewed by monalisa · 30m ago</div>
                </div>
              </div>
            </div>
          </aside>
        </div>
      </div>
    </div>
  );
}
