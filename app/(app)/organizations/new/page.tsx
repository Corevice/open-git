"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";

import { ApiClient, ApiError, type ApiFieldError } from "@/lib/api";
import { useAuth } from "@/lib/auth";

type FieldErrors = {
  login?: string;
  name?: string;
  description?: string;
  general?: string;
};

function mapApiErrors(errors: ApiFieldError[]): FieldErrors {
  const result: FieldErrors = {};
  for (const err of errors) {
    const field = (err.field ?? "").toLowerCase();
    if (field === "login") {
      result.login = err.message;
    } else if (field === "name") {
      result.name = err.message;
    } else if (field === "description") {
      result.description = err.message;
    } else {
      result.general = err.message;
    }
  }
  return result;
}

export default function NewOrganizationPage() {
  const router = useRouter();
  const { token, isAuthenticated } = useAuth();
  const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "";

  const [login, setLogin] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!login.trim() || !name.trim() || submitting) return;

    setSubmitting(true);
    setFieldErrors({});

    const client = new ApiClient(baseUrl);
    if (token) {
      client.setToken(token);
    }

    try {
      const org = await client.orgs.create({
        login: login.trim(),
        name: name.trim(),
        description: description.trim() || undefined,
      });
      router.push(`/${org.login}`);
    } catch (err) {
      if (err instanceof ApiError && err.status === 422) {
        if (err.errors && err.errors.length > 0) {
          setFieldErrors(mapApiErrors(err.errors));
        } else {
          setFieldErrors({ general: err.message });
        }
      } else if (err instanceof ApiError) {
        setFieldErrors({ general: err.message });
      } else {
        setFieldErrors({ general: "組織の作成に失敗しました。" });
      }
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
          <h1 className="mb-2 text-2xl font-semibold">新しい組織を作成</h1>
          <p className="text-sm text-[#59636e]">
            組織はチームでリポジトリを共有・管理するための名前空間です。
          </p>
        </div>

        {!isAuthenticated && (
          <p className="mb-4 text-sm text-[#cf222e]" role="alert">
            組織を作成するにはサインインが必要です。
          </p>
        )}

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label htmlFor="org-login" className="mb-1.5 block text-sm font-semibold">
              組織 ID <span className="text-[#cf222e]">*</span>
            </label>
            <input
              id="org-login"
              type="text"
              value={login}
              onChange={(e) => setLogin(e.target.value)}
              placeholder="acme-corp"
              className={`w-full rounded-md border px-3 py-2 text-sm ${
                fieldErrors.login ? "border-[#cf222e]" : "border-[#d1d9e0]"
              }`}
              required
              pattern="[a-zA-Z0-9][a-zA-Z0-9-]{1,38}"
              aria-invalid={fieldErrors.login ? true : undefined}
              aria-describedby={fieldErrors.login ? "org-login-error" : undefined}
            />
            {fieldErrors.login && (
              <p id="org-login-error" className="mt-1.5 text-[13px] text-[#cf222e]">
                {fieldErrors.login}
              </p>
            )}
          </div>

          <div className="mb-4">
            <label htmlFor="org-name" className="mb-1.5 block text-sm font-semibold">
              表示名 <span className="text-[#cf222e]">*</span>
            </label>
            <input
              id="org-name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="ACME Corporation"
              className={`w-full rounded-md border px-3 py-2 text-sm ${
                fieldErrors.name ? "border-[#cf222e]" : "border-[#d1d9e0]"
              }`}
              required
              aria-invalid={fieldErrors.name ? true : undefined}
              aria-describedby={fieldErrors.name ? "org-name-error" : undefined}
            />
            {fieldErrors.name && (
              <p id="org-name-error" className="mt-1.5 text-[13px] text-[#cf222e]">
                {fieldErrors.name}
              </p>
            )}
          </div>

          <div className="mb-4">
            <label htmlFor="org-description" className="mb-1.5 block text-sm font-semibold">
              説明 <span className="text-[color:var(--text-muted)]">(任意)</span>
            </label>
            <textarea
              id="org-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="この組織の簡単な説明"
              rows={3}
              className={`w-full rounded-md border px-3 py-2 text-sm ${
                fieldErrors.description ? "border-[#cf222e]" : "border-[#d1d9e0]"
              }`}
              aria-invalid={fieldErrors.description ? true : undefined}
              aria-describedby={fieldErrors.description ? "org-description-error" : undefined}
            />
            {fieldErrors.description && (
              <p id="org-description-error" className="mt-1.5 text-[13px] text-[#cf222e]">
                {fieldErrors.description}
              </p>
            )}
          </div>

          {fieldErrors.general && (
            <p className="mt-4 text-sm text-[#cf222e]" role="alert">
              {fieldErrors.general}
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
              disabled={!login.trim() || !name.trim() || submitting || !isAuthenticated}
              className="inline-flex items-center gap-2 rounded-md bg-[#1f883d] px-4 py-2 text-sm font-medium text-white hover:bg-[#1a7f37] disabled:cursor-not-allowed disabled:opacity-50"
            >
              {submitting && (
                <span
                  className="inline-block h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"
                  aria-hidden="true"
                />
              )}
              {submitting ? "作成中…" : "組織を作成"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
