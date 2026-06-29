"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";

import { apiClient, isApiError } from "@/lib/api-client";

const pullSchema = z
  .object({
    title: z
      .string()
      .min(1, "Title is required")
      .max(256, "Title must be 256 chars or fewer"),
    body: z.string().max(65536).optional(),
    head: z.string().min(1, "Head branch is required"),
    base: z.string().min(1, "Base branch is required"),
  })
  .refine((data) => data.head !== data.base, {
    message: "Head branch must differ from base branch",
    path: ["head"],
  });

type PullFormValues = z.infer<typeof pullSchema>;

type Props = {
  params: Promise<{ owner: string; repo: string }>;
};

export default function NewPullRequestPage({ params }: Props) {
  const router = useRouter();
  const [owner, setOwner] = useState("");
  const [repo, setRepo] = useState("");
  const [submitError, setSubmitError] = useState<string | null>(null);

  const { register, handleSubmit, formState, watch } = useForm<PullFormValues>({
    resolver: zodResolver(pullSchema),
    defaultValues: { title: "", body: "", head: "", base: "" },
  });

  const title = watch("title") ?? "";

  useEffect(() => {
    params.then(({ owner: o, repo: r }) => {
      setOwner(o);
      setRepo(r);
    });
  }, [params]);

  const onSubmit = handleSubmit(async (data) => {
    setSubmitError(null);
    try {
      const pr = await apiClient.post<{ number: number }>(
        `/api/v3/repos/${owner}/${repo}/pulls`,
        {
          title: data.title,
          body: data.body ?? "",
          head: data.head,
          base: data.base,
        },
      );
      router.push(`/${owner}/${repo}/pull/${pr.number}`);
    } catch (err) {
      if (isApiError(err)) {
        setSubmitError(err.message ?? "Failed to create pull request");
      } else {
        setSubmitError("Failed to create pull request");
      }
    }
  });

  if (!owner || !repo) return null;

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <div className="max-w-[960px] mx-auto px-6 py-6">
        <div className="mb-4">
          <Link href={`/${owner}/${repo}/pulls`} className="text-sm text-[#0969da] hover:underline">
            ← Back to pull requests
          </Link>
        </div>

        <h1 className="text-2xl font-semibold mb-6">
          New Pull Request in{" "}
          <span className="text-[#0969da]">
            {owner}/{repo}
          </span>
        </h1>

        <form onSubmit={onSubmit} className="bg-white border border-[#d0d7de] rounded-md p-6">
          {submitError && (
            <div className="mb-4 px-3 py-2 text-sm text-[#d1242f] bg-[#ffebe9] border border-[#ff8182] rounded-md">
              {submitError}
            </div>
          )}

          <div className="mb-4 grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label htmlFor="base" className="block text-sm font-semibold mb-1">
                Base branch <span className="text-[#d1242f]">*</span>
              </label>
              <input
                id="base"
                type="text"
                {...register("base")}
                className={`w-full px-3 py-2 border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#0969da] ${
                  formState.errors.base ? "border-[#d1242f]" : "border-[#d0d7de]"
                }`}
                placeholder="main"
              />
              {formState.errors.base?.message && (
                <span className="text-xs text-[#d1242f] mt-1 block">{formState.errors.base.message}</span>
              )}
            </div>

            <div>
              <label htmlFor="head" className="block text-sm font-semibold mb-1">
                Head branch <span className="text-[#d1242f]">*</span>
              </label>
              <input
                id="head"
                type="text"
                {...register("head")}
                className={`w-full px-3 py-2 border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#0969da] ${
                  formState.errors.head ? "border-[#d1242f]" : "border-[#d0d7de]"
                }`}
                placeholder="feature/my-branch"
              />
              {formState.errors.head?.message && (
                <span className="text-xs text-[#d1242f] mt-1 block">{formState.errors.head.message}</span>
              )}
            </div>
          </div>

          <div className="mb-4">
            <label htmlFor="title" className="block text-sm font-semibold mb-1">
              Title <span className="text-[#d1242f]">*</span>
            </label>
            <input
              id="title"
              type="text"
              {...register("title")}
              className={`w-full px-3 py-2 border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#0969da] ${
                formState.errors.title ? "border-[#d1242f]" : "border-[#d0d7de]"
              }`}
              placeholder="Pull request title"
              maxLength={256}
            />
            <div className="flex justify-between mt-1">
              {formState.errors.title?.message ? (
                <span className="text-xs text-[#d1242f]">{formState.errors.title.message}</span>
              ) : (
                <span className="text-xs text-[#656d76]">Required</span>
              )}
              <span className={`text-xs ${title.length > 256 ? "text-[#d1242f]" : "text-[#656d76]"}`}>
                {title.length}/256
              </span>
            </div>
          </div>

          <div className="mb-4">
            <label htmlFor="body" className="block text-sm font-semibold mb-1">
              Body
            </label>
            <textarea
              id="body"
              {...register("body")}
              className="w-full min-h-[200px] px-3 py-2 border border-[#d0d7de] rounded-md text-sm resize-y focus:outline-none focus:ring-2 focus:ring-[#0969da]"
              placeholder="Describe your changes"
            />
          </div>

          <div className="flex gap-2 justify-end">
            <Link
              href={`/${owner}/${repo}/pulls`}
              className="px-4 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white hover:bg-[#f6f8fa]"
            >
              Cancel
            </Link>
            <button
              type="submit"
              disabled={formState.isSubmitting}
              className="inline-flex items-center gap-2 px-4 py-1.5 text-sm bg-[#1f883d] text-white rounded-md font-semibold border border-black/10 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {formState.isSubmitting && <span className="animate-spin">⟳</span>}
              {formState.isSubmitting ? "Creating…" : "Create pull request"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
