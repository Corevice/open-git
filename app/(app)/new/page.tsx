"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useMemo, useState } from "react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";
const REPO_NAME_PATTERN = /^[a-zA-Z0-9._-]{1,100}$/;

interface ApiUser {
  login: string;
}

interface ApiOrg {
  login: string;
}

export default function NewRepoPage() {
  const router = useRouter();
  const [owners, setOwners] = useState<string[]>([]);
  const [owner, setOwner] = useState("");
  const [repoName, setRepoName] = useState("");
  const [description, setDescription] = useState("");
  const [visibility, setVisibility] = useState<"public" | "private">("public");
  const [initReadme, setInitReadme] = useState(true);
  const [initGitignore, setInitGitignore] = useState(false);
  const [gitignoreTemplate, setGitignoreTemplate] = useState("Node");
  const [initLicense, setInitLicense] = useState(false);
  const [license, setLicense] = useState("mit");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadOwners() {
      try {
        const headers = { Accept: "application/vnd.github+json" };
        const [userRes, orgsRes] = await Promise.all([
          fetch(`${API_BASE}/user`, { headers }),
          fetch(`${API_BASE}/user/orgs`, { headers }),
        ]);
        const logins: string[] = [];
        if (userRes.ok) {
          const user = (await userRes.json()) as ApiUser;
          logins.push(user.login);
          setOwner(user.login);
        }
        if (orgsRes.ok) {
          const orgs = (await orgsRes.json()) as ApiOrg[];
          orgs.forEach((o) => logins.push(o.login));
        }
        setOwners(logins);
      } catch {
        setOwners(["octocat"]);
        setOwner("octocat");
      }
    }
    loadOwners();
  }, []);

  const nameValidation = useMemo(() => {
    const trimmed = repoName.trim();
    if (!trimmed) return { valid: false, message: "リポジトリ名を入力してください。" };
    if (!REPO_NAME_PATTERN.test(trimmed)) {
      return {
        valid: false,
        message:
          "名前は英数字・ピリオド・ハイフン・アンダースコアのみ、1〜100文字で指定してください。",
      };
    }
    return {
      valid: true,
      message: `✓ 利用可能な名前です。 ${owner}/${trimmed}`,
    };
  }, [repoName, owner]);

  const canSubmit = nameValidation.valid && owner && !submitting;

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;

    setSubmitting(true);
    setError(null);

    const body: Record<string, unknown> = {
      name: repoName.trim(),
      description: description.trim() || undefined,
      private: visibility === "private",
      auto_init: initReadme,
    };

    if (initGitignore && gitignoreTemplate !== "None") {
      body.gitignore_template = gitignoreTemplate.toLowerCase();
    }
    if (initLicense && license !== "none") {
      body.license_template = license;
    }

    try {
      const res = await fetch(`${API_BASE}/user/repos`, {
        method: "POST",
        headers: {
          Accept: "application/vnd.github+json",
          "Content-Type": "application/json",
        },
        body: JSON.stringify(body),
      });

      if (res.status === 201) {
        const created = (await res.json()) as { owner: { login: string }; name: string };
        router.push(`/${created.owner.login}/${created.name}`);
        return;
      }

      const err = await res.json().catch(() => ({}));
      setError(
        (err as { message?: string }).message ??
          `リポジトリの作成に失敗しました (${res.status})`,
      );
    } catch {
      setError("ネットワークエラーが発生しました。");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <header className="sticky top-0 z-50 flex h-16 items-center justify-between border-b border-[#d1d9e0] bg-white/85 px-6 backdrop-blur">
        <Link href="/dashboard" className="flex items-center gap-2 text-lg font-extrabold">
          <span className="text-xl">🐙</span>
          <span>OpenHub</span>
        </Link>
        <Link href="/dashboard" className="rounded-md px-3 py-1.5 text-sm hover:bg-[#f6f8fa]">
          ダッシュボード
        </Link>
      </header>

      <div className="mx-auto max-w-[640px] px-6 py-8">
        <div className="mb-6 border-b border-[#d1d9e0] pb-4">
          <h1 className="mb-2 text-2xl font-semibold">新しいリポジトリを作成</h1>
          <p className="text-sm text-[#59636e]">
            リポジトリにはプロジェクトのすべてのファイルとリビジョン履歴が含まれます。
          </p>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <div className="grid grid-cols-[1fr_auto_2fr] items-end gap-3">
              <div>
                <label htmlFor="owner" className="mb-1.5 block text-sm font-semibold">
                  オーナー <span className="text-[#cf222e]">*</span>
                </label>
                <select
                  id="owner"
                  value={owner}
                  onChange={(e) => setOwner(e.target.value)}
                  className="w-full rounded-md border border-[#d1d9e0] bg-white px-3 py-2 text-sm"
                  required
                >
                  {owners.map((o) => (
                    <option key={o} value={o}>
                      {o}
                    </option>
                  ))}
                </select>
              </div>
              <div className="pb-2 text-2xl text-[#59636e] select-none">/</div>
              <div>
                <label htmlFor="repo-name" className="mb-1.5 block text-sm font-semibold">
                  リポジトリ名 <span className="text-[#cf222e]">*</span>
                </label>
                <input
                  id="repo-name"
                  type="text"
                  value={repoName}
                  onChange={(e) => setRepoName(e.target.value)}
                  placeholder="hello-world"
                  className="w-full rounded-md border border-[#d1d9e0] px-3 py-2 text-sm"
                  required
                  pattern="[a-zA-Z0-9._-]{1,100}"
                />
              </div>
            </div>
            <div
              className={`mt-1.5 text-[13px] ${nameValidation.valid ? "text-[#1f883d]" : "text-[#cf222e]"}`}
            >
              {nameValidation.message}
            </div>
          </div>

          <div className="mb-4">
            <label htmlFor="description" className="mb-1.5 block text-sm font-semibold">
              説明 <span className="text-[color:var(--text-muted)]">(任意)</span>
            </label>
            <textarea
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="このリポジトリの簡単な説明"
              rows={3}
              className="w-full rounded-md border border-[#d1d9e0] px-3 py-2 text-sm"
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
              <div>
                <div className="text-sm font-semibold">Public</div>
                <div className="text-[13px] text-[#59636e]">誰でもこのリポジトリを閲覧できます。</div>
              </div>
            </label>
            <label className="flex cursor-pointer items-start gap-3 rounded-md border border-[#d1d9e0] p-3 hover:bg-[#f6f8fa]">
              <input
                type="radio"
                name="visibility"
                checked={visibility === "private"}
                onChange={() => setVisibility("private")}
                className="mt-1"
              />
              <div>
                <div className="text-sm font-semibold">Private</div>
                <div className="text-[13px] text-[#59636e]">アクセス権を持つユーザーのみ閲覧できます。</div>
              </div>
            </label>
          </div>

          <hr className="my-6 border-t border-[#d1d9e0]" />

          <div className="rounded-md border border-[#d1d9e0] bg-[#f6f8fa] p-4">
            <h3 className="mb-3 text-sm font-semibold">このリポジトリを初期化</h3>
            <label className="flex items-start gap-2.5 py-2">
              <input
                type="checkbox"
                checked={initReadme}
                onChange={(e) => setInitReadme(e.target.checked)}
                className="mt-1"
              />
              <span className="text-sm">README ファイルを追加</span>
            </label>
            <label className="flex items-start gap-2.5 py-2">
              <input
                type="checkbox"
                checked={initGitignore}
                onChange={(e) => setInitGitignore(e.target.checked)}
                className="mt-1"
              />
              <span className="text-sm">.gitignore を追加</span>
            </label>
            {initGitignore && (
              <select
                value={gitignoreTemplate}
                onChange={(e) => setGitignoreTemplate(e.target.value)}
                className="ml-7 mb-2 w-[calc(100%-1.75rem)] rounded-md border border-[#d1d9e0] bg-white px-3 py-2 text-sm"
              >
                <option>None</option>
                <option>Node</option>
                <option>Python</option>
                <option>Go</option>
                <option>Rust</option>
              </select>
            )}
            <label className="flex items-start gap-2.5 py-2">
              <input
                type="checkbox"
                checked={initLicense}
                onChange={(e) => setInitLicense(e.target.checked)}
                className="mt-1"
              />
              <span className="text-sm">ライセンスを選択</span>
            </label>
            {initLicense && (
              <select
                value={license}
                onChange={(e) => setLicense(e.target.value)}
                className="ml-7 w-[calc(100%-1.75rem)] rounded-md border border-[#d1d9e0] bg-white px-3 py-2 text-sm"
              >
                <option value="mit">MIT License</option>
                <option value="apache-2.0">Apache License 2.0</option>
                <option value="gpl-3.0">GNU GPL v3.0</option>
              </select>
            )}
          </div>

          {error && (
            <p className="mt-4 text-sm text-[#cf222e]" role="alert">
              {error}
            </p>
          )}

          <div className="mt-6 flex justify-end gap-2">
            <Link
              href="/dashboard"
              className="rounded-md border border-[#d1d9e0] bg-white px-4 py-2 text-sm hover:bg-[#f6f8fa]"
            >
              キャンセル
            </Link>
            <button
              type="submit"
              disabled={!canSubmit}
              className="rounded-md bg-[#1f883d] px-4 py-2 text-sm font-medium text-white hover:bg-[#1a7f37] disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {submitting ? "作成中…" : "リポジトリを作成"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
