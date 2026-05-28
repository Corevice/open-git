"use client";

import Link from "next/link";
import { useState, FormEvent } from "react";

export default function RepoCreatePage() {
  const [owner, setOwner] = useState("octocat");
  const [repoName, setRepoName] = useState("awesome-project");
  const [description, setDescription] = useState("");
  const [visibility, setVisibility] = useState("public");
  const [initReadme, setInitReadme] = useState(false);
  const [initGitignore, setInitGitignore] = useState(false);
  const [gitignoreTemplate, setGitignoreTemplate] = useState("None");
  const [initLicense, setInitLicense] = useState(false);
  const [license, setLicense] = useState("None");

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <header className="app-bar flex h-16 items-center justify-between border-b border-[#d1d9e0] bg-white/85 px-6 sticky top-0 z-50 backdrop-blur">
        <div className="flex items-center gap-2 text-lg font-extrabold">
          <span className="text-xl">🐙</span>
          <span>OpenHub</span>
        </div>
        <div className="flex items-center gap-3">
          <Link href="/05-repo-list" className="text-sm text-[color:var(--text-secondary)] hover:text-[color:var(--primary)] px-2 py-1">リポジトリ</Link>
          <Link href="/05-repo-list" className="text-sm text-[color:var(--text-secondary)] hover:text-[color:var(--primary)] px-2 py-1">マイページ</Link>
        </div>
      </header>

      <div className="mx-auto max-w-[640px] px-6 py-8">
        <div className="mb-6 border-b border-[#d1d9e0] pb-4">
          <h1 className="mb-2 text-2xl font-semibold">新しいリポジトリを作成</h1>
          <p className="m-0 text-sm text-[#59636e]">
            リポジトリには、プロジェクトのすべてのファイル、リビジョン履歴が含まれます。既存のリポジトリを別の場所にお持ちですか？{" "}
            <Link href="/06-repo-create" className="text-[#0969da] hover:underline">インポートできます</Link>。
          </p>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <div className="grid grid-cols-[1fr_auto_2fr] items-end gap-3">
              <div>
                <label className="mb-1.5 block text-sm font-semibold" htmlFor="owner">
                  オーナー <span className="text-[#cf222e]">*</span>
                </label>
                <select
                  id="owner"
                  value={owner}
                  onChange={(e) => setOwner(e.target.value)}
                  className="w-full rounded-md border border-[#d1d9e0] bg-white px-3 py-2 text-sm focus:border-[#0969da] focus:outline-none focus:ring-[3px] focus:ring-[#0969da]/15"
                >
                  <option>octocat</option>
                  <option>open-source-org</option>
                  <option>my-team</option>
                </select>
              </div>
              <div className="select-none pb-2 text-2xl text-[#59636e]">/</div>
              <div>
                <label className="mb-1.5 block text-sm font-semibold" htmlFor="repo-name">
                  リポジトリ名 <span className="text-[#cf222e]">*</span>
                </label>
                <input
                  id="repo-name"
                  type="text"
                  value={repoName}
                  onChange={(e) => setRepoName(e.target.value)}
                  placeholder="例: hello-world"
                  className="w-full rounded-md border border-[#d1d9e0] px-3 py-2 text-sm focus:border-[#0969da] focus:outline-none focus:ring-[3px] focus:ring-[#0969da]/15"
                />
              </div>
            </div>
            <div className="mt-1.5 text-[13px] text-[#1f883d]">
              ✓ 利用可能な名前です。 <span className="font-mono text-[#59636e]">{owner}/{repoName}</span>
            </div>
            <div className="mt-1 text-xs text-[#59636e]">
              短く覚えやすい名前をおすすめします。何かインスピレーションが必要ですか？ <strong>fluffy-octo-spoon</strong> はいかがでしょう？
            </div>
          </div>

          <div className="mb-4">
            <label className="mb-1.5 block text-sm font-semibold" htmlFor="description">
              説明 <span className="text-[#59636e]">(任意)</span>
            </label>
            <input
              id="description"
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="このリポジトリの簡単な説明"
              className="w-full rounded-md border border-[#d1d9e0] px-3 py-2 text-sm focus:border-[#0969da] focus:outline-none focus:ring-[3px] focus:ring-[#0969da]/15"
            />
          </div>

          <hr className="my-6 border-t border-[#d1d9e0]" />

          <div className="mb-4">
            <label className="mb-1.5 block text-sm font-semibold">
              公開範囲 <span className="text-[#cf222e]">*</span>
            </label>

            <label className="mb-2 flex cursor-pointer items-start gap-3 rounded-md border border-[#d1d9e0] p-3 hover:bg-[#f6f8fa]">
              <input
                type="radio"
                name="visibility"
                checked={visibility === "public"}
                onChange={() => setVisibility("public")}
                className="mt-1"
              />
              <span className="text-xl">📖</span>
              <div className="flex-1">
                <div className="mb-0.5 text-sm font-semibold">Public</div>
                <div className="text-[13px] text-[#59636e]">インターネット上の誰でもこのリポジトリを見ることができます。コミットできる人はあなたが指定します。</div>
              </div>
            </label>

            <label className="mb-2 flex cursor-pointer items-start gap-3 rounded-md border border-[#d1d9e0] p-3 hover:bg-[#f6f8fa]">
              <input
                type="radio"
                name="visibility"
                checked={visibility === "private"}
                onChange={() => setVisibility("private")}
                className="mt-1"
              />
              <span className="text-xl">🔒</span>
              <div className="flex-1">
                <div className="mb-0.5 text-sm font-semibold">Private</div>
                <div className="text-[13px] text-[#59636e]">アクセス権を付与した人だけがこのリポジトリを見たり、コミットできたりします。</div>
              </div>
            </label>
          </div>

          <hr className="my-6 border-t border-[#d1d9e0]" />

          <div className="rounded-md border border-[#d1d9e0] bg-[#f6f8fa] p-4">
            <h3 className="mb-1 text-sm font-semibold">このリポジトリを初期化</h3>
            <p className="mb-4 text-[13px] text-[#59636e]">これにより、すぐにリポジトリをクローンできるようになります。後でスキップすることもできます。</p>

            <div className="flex items-start gap-2.5 py-2">
              <input
                type="checkbox"
                id="init-readme"
                checked={initReadme}
                onChange={(e) => setInitReadme(e.target.checked)}
                className="mt-1"
              />
              <div>
                <label className="text-sm font-medium" htmlFor="init-readme">README ファイルを追加</label>
                <div className="mt-0.5 text-[13px] text-[#59636e]">
                  プロジェクトの長い説明を書ける場所です。 <Link href="/06-repo-create" className="text-[#0969da] hover:underline">README について</Link>。
                </div>
              </div>
            </div>

            <div className="flex items-start gap-2.5 py-2">
              <input
                type="checkbox"
                id="init-gitignore"
                checked={initGitignore}
                onChange={(e) => setInitGitignore(e.target.checked)}
                className="mt-1"
              />
              <div>
                <label className="text-sm font-medium" htmlFor="init-gitignore">.gitignore を追加</label>
                <div className="mt-0.5 text-[13px] text-[#59636e]">
                  追跡しないファイルをテンプレートから選択します。 <Link href="/06-repo-create" className="text-[#0969da] hover:underline">.gitignore について</Link>。
                </div>
              </div>
            </div>

            <div className="mb-3 ml-[30px] mt-2">
              <label className="mb-1.5 block text-sm font-semibold" htmlFor="gitignore-template">.gitignore テンプレート</label>
              <select
                id="gitignore-template"
                value={gitignoreTemplate}
                onChange={(e) => setGitignoreTemplate(e.target.value)}
                className="w-full rounded-md border border-[#d1d9e0] bg-white px-3 py-2 text-sm focus:border-[#0969da] focus:outline-none focus:ring-[3px] focus:ring-[#0969da]/15"
              >
                <option>None</option>
                <option>Node</option>
                <option>Python</option>
                <option>Java</option>
                <option>Go</option>
              </select>
            </div>

            <div className="flex items-start gap-2.5 py-2">
              <input
                type="checkbox"
                id="init-license"
                checked={initLicense}
                onChange={(e) => setInitLicense(e.target.checked)}
                className="mt-1"
              />
              <div>
                <label className="text-sm font-medium" htmlFor="init-license">ライセンスを選択</label>
                <div className="mt-0.5 text-[13px] text-[#59636e]">
                  ライセンスは、他者があなたのコードで何ができて何ができないかを伝えるものです。 <Link href="/06-repo-create" className="text-[#0969da] hover:underline">ライセンスについて</Link>。
                </div>
              </div>
            </div>

            <div className="ml-[30px] mt-2">
              <label className="mb-1.5 block text-sm font-semibold" htmlFor="license">ライセンス</label>
              <select
                id="license"
                value={license}
                onChange={(e) => setLicense(e.target.value)}
                className="w-full rounded-md border border-[#d1d9e0] bg-white px-3 py-2 text-sm focus:border-[#0969da] focus:outline-none focus:ring-[3px] focus:ring-[#0969da]/15"
              >
                <option>None</option>
                <option>MIT License</option>
                <option>Apache License 2.0</option>
                <option>GNU GPL v3.0</option>
                <option>BSD 3-Clause</option>
              </select>
            </div>
          </div>

          <hr className="my-6 border-t border-[#d1d9e0]" />

          <div className="flex justify-end gap-2 pt-2">
            <Link
              href="/05-repo-list"
              className="rounded-md border border-[#d1d9e0] bg-white px-4 py-2 text-sm font-medium text-[color:var(--text-primary)] hover:bg-[#f6f8fa]"
            >
              キャンセル
            </Link>
            <button
              type="submit"
              className="rounded-md bg-[#1f883d] px-4 py-2 text-sm font-medium text-white hover:bg-[#1a7335]"
            >
              リポジトリを作成
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
