"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useMemo, useState } from "react";

import { ApiClient, ApiError } from "@/lib/api";
import type { OrgProfile } from "@/lib/api-types";
import { useAuth } from "@/lib/auth";

type FieldErrors = Record<string, string[]>;

function isApiError(err: unknown): err is ApiError {
  return err instanceof ApiError;
}

function getFieldErrors(err: unknown): FieldErrors {
  if (typeof err !== "object" || err === null || !("field_errors" in err)) {
    return {};
  }
  const fieldErrors = (err as { field_errors?: FieldErrors }).field_errors;
  return fieldErrors ?? {};
}

const GITIGNORE_OPTIONS = [
  { value: "", label: "なし" },
  { value: "Node", label: "Node" },
  { value: "Python", label: "Python" },
  { value: "Go", label: "Go" },
] as const;

const LICENSE_OPTIONS = [
  { value: "", label: "なし" },
  { value: "mit", label: "MIT" },
  { value: "apache-2.0", label: "Apache-2.0" },
  { value: "gpl-3.0", label: "GPL-3.0" },
] as const;

export default function NewRepoPage() {
  const router = useRouter();
  const { isAuthenticated, token } = useAuth();
  const [userLogin, setUserLogin] = useState<string>("");
  const [orgs, setOrgs] = useState<OrgProfile[]>([]);
  const [selectedOwner, setSelectedOwner] = useState("");
  const [repoName, setRepoName] = useState("");
  const [description, setDescription] = useState("");
  const [visibility, setVisibility] = useState<"public" | "private">("public");
  const [autoInit, setAutoInit] = useState(false);
  const [gitignoreTemplate, setGitignoreTemplate] = useState("");
  const [licenseTemplate, setLicenseTemplate] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});

  const apiClient = useMemo(
    () =>
      new ApiClient(
        process.env.NEXT_PUBLIC_API_BASE_URL ??
          process.env.NEXT_PUBLIC_API_URL ??
          "http://localhost:8080",
      ),
    [],
  );

  useEffect(() => {
    if (token) {
      apiClient.setToken(token);
    }
  }, [apiClient, token]);

  useEffect(() => {
    if (!isAuthenticated || !token) {
      return;
    }

    let cancelled = false;

    async function loadOwners() {
      try {
        const [user, userOrgs] = await Promise.all([
          apiClient.users.getCurrent(),
          apiClient.get<OrgProfile[]>("/api/v3/user/orgs"),
        ]);
        if (cancelled) {
          return;
        }
        setUserLogin(user.login);
        setOrgs(userOrgs);
        setSelectedOwner((prev) => prev || user.login);
      } catch {
        if (!cancelled) {
          setError("オーナー情報の取得に失敗しました。");
        }
      }
    }

    void loadOwners();

    return () => {
      cancelled = true;
    };
  }, [apiClient, isAuthenticated, token]);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!repoName.trim() || submitting || !selectedOwner) return;

    setSubmitting(true);
    setError(null);
    setFieldErrors({});

    const body = {
      name: repoName.trim(),
      description: description.trim() || undefined,
      private: visibility === "private",
      auto_init: autoInit,
      ...(gitignoreTemplate ? { gitignore_template: gitignoreTemplate } : {}),
      ...(licenseTemplate ? { license_template: licenseTemplate } : {}),
    };

    try {
      const created =
        selectedOwner === userLogin
          ? await apiClient.repos.createForUser(body)
          : await apiClient.repos.createForOrg(selectedOwner, body);
      router.push(`/${created.owner}/${created.name}`);
    } catch (err) {
      if (isApiError(err) && err.status === 422) {
        setFieldErrors(getFieldErrors(err));
        if (Object.keys(getFieldErrors(err)).length === 0) {
          setError(err.message || "入力内容に問題があります。");
        }
      } else if (isApiError(err)) {
        setError(err.message || "リポジトリの作成に失敗しました。");
      } else {
        setError("ネットワークエラーが発生しました。");
      }
    } finally {
      setSubmitting(false);
    }
  };

  const nameError = fieldErrors.name?.[0];
  const visibilityError = fieldErrors.visibility?.[0];
  const descriptionError = fieldErrors.description?.[0];

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

        {!isAuthenticated && (
          <p className="mb-4 text-sm text-[#cf222e]" role="alert">
            リポジトリを作成するにはサインインが必要です。
          </p>
        )}

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label htmlFor="owner" className="mb-1.5 block text-sm font-semibold">
              オーナー <span className="text-[#cf222e]">*</span>
            </label>
            <select
              id="owner"
              value={selectedOwner}
              onChange={(e) => setSelectedOwner(e.target.value)}
              className="w-full rounded-md border border-[#d1d9e0] px-3 py-2 text-sm"
              required
              disabled={!userLogin}
            >
              {userLogin && (
                <option value={userLogin}>{userLogin} (ユーザー)</option>
              )}
              {orgs.map((org) => (
                <option key={org.login} value={org.login}>
                  {org.login} (組織)
                </option>
              ))}
            </select>
          </div>

          <div className="mb-4">
            <label htmlFor="repo-name" className="mb-1.5 block text-sm font-semibold">
              リポジトリ名 <span className="text-[#cf222e]">*</span>
            </label>
            <input
              id="repo-name"
              type="text"
              value={repoName}
              onChange={(e) => setRepoName(e.target.value)}
              placeholder="hello-world"
              className={`w-full rounded-md border px-3 py-2 text-sm ${
                nameError ? "border-[#cf222e]" : "border-[#d1d9e0]"
              }`}
              required
              pattern="[a-zA-Z0-9._-]{1,100}"
              aria-invalid={nameError ? true : undefined}
              aria-describedby={nameError ? "repo-name-error" : undefined}
            />
            {nameError && (
              <p id="repo-name-error" className="mt-1.5 text-[13px] text-[#cf222e]">
                {nameError}
              </p>
            )}
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
              className={`w-full rounded-md border px-3 py-2 text-sm ${
                descriptionError ? "border-[#cf222e]" : "border-[#d1d9e0]"
              }`}
              aria-invalid={descriptionError ? true : undefined}
              aria-describedby={descriptionError ? "description-error" : undefined}
            />
            {descriptionError && (
              <p id="description-error" className="mt-1.5 text-[13px] text-[#cf222e]">
                {descriptionError}
              </p>
            )}
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
            {visibilityError && (
              <p className="mt-1.5 text-[13px] text-[#cf222e]">{visibilityError}</p>
            )}
          </div>

          <hr className="my-6 border-t border-[#d1d9e0]" />

          <div className="mb-4">
            <label className="mb-1.5 block text-sm font-semibold">初期化オプション</label>
            <label className="mb-3 flex cursor-pointer items-center gap-2">
              <input
                type="checkbox"
                checked={autoInit}
                onChange={(e) => setAutoInit(e.target.checked)}
              />
              <span className="text-sm">README でこのリポジトリを初期化する</span>
            </label>
            <div className="mb-3">
              <label htmlFor="gitignore-template" className="mb-1.5 block text-sm">
                .gitignore テンプレート
              </label>
              <select
                id="gitignore-template"
                value={gitignoreTemplate}
                onChange={(e) => setGitignoreTemplate(e.target.value)}
                className="w-full rounded-md border border-[#d1d9e0] px-3 py-2 text-sm"
              >
                {GITIGNORE_OPTIONS.map((opt) => (
                  <option key={opt.value || "none"} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label htmlFor="license-template" className="mb-1.5 block text-sm">
                ライセンス
              </label>
              <select
                id="license-template"
                value={licenseTemplate}
                onChange={(e) => setLicenseTemplate(e.target.value)}
                className="w-full rounded-md border border-[#d1d9e0] px-3 py-2 text-sm"
              >
                {LICENSE_OPTIONS.map((opt) => (
                  <option key={opt.value || "none"} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>
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
              disabled={
                !repoName.trim() ||
                submitting ||
                !isAuthenticated ||
                !selectedOwner
              }
              className="inline-flex items-center gap-2 rounded-md bg-[#1f883d] px-4 py-2 text-sm font-medium text-white hover:bg-[#1a7f37] disabled:cursor-not-allowed disabled:opacity-50"
            >
              {submitting && (
                <span
                  className="inline-block h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"
                  aria-hidden="true"
                />
              )}
              {submitting ? "作成中…" : "リポジトリを作成"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
