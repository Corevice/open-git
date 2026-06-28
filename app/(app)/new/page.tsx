"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";

import { apiClient } from "@/lib/api-client";
import { useAuth } from "@/lib/auth";
import { ApiError } from "@/lib/api";
import { repoNameSchema } from "@/lib/validations";

type CreateRepoResponse = {
  owner: string;
  name: string;
};

type FieldErrors = Record<string, string[]>;

const newRepoFormSchema = z.object({
  name: repoNameSchema,
  description: z.string().optional(),
  visibility: z.enum(["public", "private"]),
  auto_init: z.boolean().default(false),
});

type NewRepoFormValues = z.infer<typeof newRepoFormSchema>;

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

export default function NewRepoPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuth();
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});

  const { register, handleSubmit, formState, watch, setValue } = useForm<NewRepoFormValues>({
    resolver: zodResolver(newRepoFormSchema),
    defaultValues: { name: "", description: "", visibility: "public", auto_init: false },
  });

  const visibility = watch("visibility");

  const onSubmit = handleSubmit(async (values) => {
    if (!isAuthenticated || formState.isSubmitting) return;

    setError(null);
    setFieldErrors({});

    try {
      const created = (await apiClient.createRepo(
        values.name.trim(),
        values.visibility,
        {
          description: values.description?.trim() || undefined,
          autoInit: values.auto_init,
        },
      )) as CreateRepoResponse;
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
    }
  });

  const nameError = formState.errors.name?.message ?? fieldErrors.name?.[0];
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

        <form onSubmit={onSubmit}>
          <div className="mb-4">
            <label htmlFor="repo-name" className="mb-1.5 block text-sm font-semibold">
              リポジトリ名 <span className="text-[#cf222e]">*</span>
            </label>
            <input
              id="repo-name"
              type="text"
              {...register("name")}
              placeholder="hello-world"
              className={`w-full rounded-md border px-3 py-2 text-sm ${
                nameError ? "border-[#cf222e]" : "border-[#d1d9e0]"
              }`}
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
              {...register("description")}
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
                onChange={() => setValue("visibility", "public")}
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
                onChange={() => setValue("visibility", "private")}
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

          <div className="mb-4">
            <label className="flex cursor-pointer items-start gap-3 rounded-md border border-[#d1d9e0] p-3 hover:bg-[#f6f8fa]">
              <input type="checkbox" {...register("auto_init")} className="mt-1" />
              <div>
                <div className="text-sm font-semibold">リポジトリをREADMEで初期化する</div>
                <div className="text-[13px] text-[#59636e]">
                  すぐにクローンできるリポジトリを作成します
                </div>
              </div>
            </label>
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
              disabled={formState.isSubmitting || !isAuthenticated}
              className="inline-flex items-center gap-2 rounded-md bg-[#1f883d] px-4 py-2 text-sm font-medium text-white hover:bg-[#1a7f37] disabled:cursor-not-allowed disabled:opacity-50"
            >
              {formState.isSubmitting && (
                <span
                  className="inline-block h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"
                  aria-hidden="true"
                />
              )}
              {formState.isSubmitting ? "作成中…" : "リポジトリを作成"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
