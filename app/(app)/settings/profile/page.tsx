"use client";

import Link from "next/link";
import { FormEvent, useEffect, useMemo, useState } from "react";

import { ApiClient, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";

type FieldErrors = {
  name?: string;
  email?: string;
  bio?: string;
  avatar_url?: string;
  general?: string;
};

type ApiFieldError = {
  field?: string;
  resource?: string;
  message: string;
};

type ApiErrorBody = {
  message?: string;
  errors?: ApiFieldError[];
};

function mapErrorsToFields(errors: ApiFieldError[]): FieldErrors {
  const result: FieldErrors = {};
  for (const err of errors) {
    const field = (err.field ?? "").toLowerCase();
    if (field === "name") {
      result.name = err.message;
    } else if (field === "email") {
      result.email = err.message;
    } else if (field === "bio") {
      result.bio = err.message;
    } else if (field === "avatar_url") {
      result.avatar_url = err.message;
    } else {
      result.general = err.message;
    }
  }
  return result;
}

async function fetchUpdateErrorBody(
  baseURL: string,
  token: string,
  data: { name: string; email: string; bio: string; avatar_url: string },
): Promise<ApiErrorBody | null> {
  try {
    const response = await fetch(`${baseURL}/api/v3/user`, {
      method: "PATCH",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(data),
    });
    return (await response.json()) as ApiErrorBody;
  } catch {
    return null;
  }
}

export default function ProfileSettingsPage() {
  const { token } = useAuth();
  const baseURL =
    process.env.NEXT_PUBLIC_API_BASE_URL ??
    process.env.NEXT_PUBLIC_API_URL ??
    "http://localhost:8080";

  const apiClient = useMemo(() => new ApiClient(baseURL), [baseURL]);

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [bio, setBio] = useState("");
  const [avatarUrl, setAvatarUrl] = useState("");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [successMessage, setSuccessMessage] = useState("");

  useEffect(() => {
    if (token) {
      apiClient.setToken(token);
    }
  }, [apiClient, token]);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      try {
        const user = await apiClient.users.getCurrent();
        if (!cancelled) {
          setName(user.name ?? "");
          setEmail(user.email ?? "");
          setBio(user.bio ?? "");
          setAvatarUrl(user.avatar_url ?? "");
        }
      } catch (err) {
        if (!cancelled) {
          setFieldErrors({
            general:
              err instanceof Error
                ? err.message
                : "プロフィールの読み込みに失敗しました。",
          });
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [apiClient]);

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setFieldErrors({});
    setSuccessMessage("");
    setSubmitting(true);

    const payload = {
      name: name.trim(),
      email: email.trim(),
      bio: bio.trim(),
      avatar_url: avatarUrl.trim(),
    };

    try {
      await apiClient.users.updateCurrent(payload);
      setSuccessMessage("プロフィールを更新しました。");
    } catch (err) {
      if (err instanceof ApiError && err.status === 422 && token) {
        const body = await fetchUpdateErrorBody(baseURL, token, payload);
        if (body?.errors && body.errors.length > 0) {
          setFieldErrors(mapErrorsToFields(body.errors));
          return;
        }
        setFieldErrors({ general: err.message });
        return;
      }

      setFieldErrors({
        general:
          err instanceof ApiError
            ? err.message
            : "プロフィールの更新に失敗しました。",
      });
    } finally {
      setSubmitting(false);
    }
  };

  const inputClass = (field: keyof FieldErrors) =>
    `w-full rounded-md border px-3 py-2 text-sm ${
      fieldErrors[field] ? "border-[#cf222e]" : "border-[#d1d9e0]"
    }`;

  if (loading) {
    return (
      <main className="mx-auto max-w-[640px] px-6 py-8">
        <p className="text-sm text-[#59636e]">読み込み中…</p>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-[640px] px-6 py-8">
      <div className="mb-6 border-b border-[#d1d9e0] pb-4">
        <h1 className="mb-2 text-2xl font-semibold">プロフィール設定</h1>
        <p className="text-sm text-[#59636e]">
          表示名、メールアドレス、アバターなどのアカウント情報を更新します。
        </p>
      </div>

      <nav className="mb-6 flex gap-4 text-sm">
        <span className="font-semibold text-[#0969da]">プロフィール</span>
        <Link href="/settings/tokens" className="text-[#0969da] hover:underline">
          Personal Access Tokens
        </Link>
      </nav>

      <form onSubmit={handleSubmit}>
        <div className="mb-4">
          <label htmlFor="name" className="mb-1.5 block text-sm font-semibold">
            表示名
          </label>
          <input
            id="name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className={inputClass("name")}
            aria-invalid={fieldErrors.name ? true : undefined}
          />
          {fieldErrors.name && (
            <p className="mt-1.5 text-[13px] text-[#cf222e]" role="alert">
              {fieldErrors.name}
            </p>
          )}
        </div>

        <div className="mb-4">
          <label htmlFor="email" className="mb-1.5 block text-sm font-semibold">
            メールアドレス
          </label>
          <input
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className={inputClass("email")}
            aria-invalid={fieldErrors.email ? true : undefined}
          />
          {fieldErrors.email && (
            <p className="mt-1.5 text-[13px] text-[#cf222e]" role="alert">
              {fieldErrors.email}
            </p>
          )}
        </div>

        <div className="mb-4">
          <label htmlFor="avatar_url" className="mb-1.5 block text-sm font-semibold">
            アバター URL
          </label>
          <input
            id="avatar_url"
            type="url"
            value={avatarUrl}
            onChange={(e) => setAvatarUrl(e.target.value)}
            placeholder="https://example.com/avatar.png"
            className={inputClass("avatar_url")}
            aria-invalid={fieldErrors.avatar_url ? true : undefined}
          />
          {fieldErrors.avatar_url && (
            <p className="mt-1.5 text-[13px] text-[#cf222e]" role="alert">
              {fieldErrors.avatar_url}
            </p>
          )}
        </div>

        <div className="mb-4">
          <label htmlFor="bio" className="mb-1.5 block text-sm font-semibold">
            自己紹介
          </label>
          <textarea
            id="bio"
            value={bio}
            onChange={(e) => setBio(e.target.value)}
            rows={4}
            className={inputClass("bio")}
            aria-invalid={fieldErrors.bio ? true : undefined}
          />
          {fieldErrors.bio && (
            <p className="mt-1.5 text-[13px] text-[#cf222e]" role="alert">
              {fieldErrors.bio}
            </p>
          )}
        </div>

        {fieldErrors.general && (
          <p className="mb-4 text-sm text-[#cf222e]" role="alert">
            {fieldErrors.general}
          </p>
        )}

        {successMessage && (
          <p className="mb-4 text-sm text-[#1a7f37]" role="status">
            {successMessage}
          </p>
        )}

        <div className="flex justify-end gap-2">
          <Link
            href="/dashboard"
            className="rounded-md border border-[#d1d9e0] bg-white px-4 py-2 text-sm hover:bg-[#f6f8fa]"
          >
            キャンセル
          </Link>
          <button
            type="submit"
            disabled={submitting}
            className="rounded-md bg-[#1f883d] px-4 py-2 text-sm font-medium text-white hover:bg-[#1a7f37] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {submitting ? "保存中…" : "変更を保存"}
          </button>
        </div>
      </form>
    </main>
  );
}
