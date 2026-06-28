"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

import { ApiError, createPullRequest } from "@/lib/api";
import { renderMarkdown } from "@/lib/markdown";

type Branch = { name: string };

type Props = {
  params: Promise<{ owner: string; repo: string }>;
};

export default function NewPullRequestPage({ params }: Props) {
  const router = useRouter();
  const [owner, setOwner] = useState("");
  const [repo, setRepo] = useState("");
  const [branches, setBranches] = useState<Branch[]>([]);
  const [base, setBase] = useState("");
  const [head, setHead] = useState("");
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [showPreview, setShowPreview] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    params.then(({ owner: o, repo: r }) => {
      setOwner(o);
      setRepo(r);
    });
  }, [params]);

  useEffect(() => {
    if (!owner || !repo) return;
    fetch(`/repos/${owner}/${repo}/branches?per_page=100`)
      .then((res) => (res.ok ? res.json() : []))
      .then((data: Branch[]) => {
        setBranches(data);
        if (data.length > 0) {
          const defaultBranch = data.find((b) => b.name === "main") ?? data[0];
          setBase(defaultBranch.name);
          setHead(data.length > 1 ? data[1].name : defaultBranch.name);
        }
      })
      .catch(() => setBranches([]));
  }, [owner, repo]);

  const branchError = base && head && base === head ? "Base and head branches must be different" : null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim()) {
      setError("Title is required");
      return;
    }
    if (branchError) {
      setError(branchError);
      return;
    }

    setSubmitting(true);
    setError(null);
    try {
      const created = await createPullRequest(owner, repo, {
        title: title.trim(),
        head,
        base,
        body,
      });
      router.push(`/${owner}/${repo}/pull/${created.number}`);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError(err instanceof Error ? err.message : "Failed to create pull request");
      }
      setSubmitting(false);
    }
  };

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
          Open a pull request in{" "}
          <span className="text-[#0969da]">
            {owner}/{repo}
          </span>
        </h1>

        <form onSubmit={handleSubmit} className="bg-white border border-[#d0d7de] rounded-md p-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
            <div>
              <label htmlFor="base" className="block text-sm font-semibold mb-1">
                Base branch
              </label>
              <select
                id="base"
                value={base}
                onChange={(e) => setBase(e.target.value)}
                className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm bg-white"
              >
                {branches.map((branch) => (
                  <option key={branch.name} value={branch.name}>
                    {branch.name}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label htmlFor="head" className="block text-sm font-semibold mb-1">
                Compare branch
              </label>
              <select
                id="head"
                value={head}
                onChange={(e) => setHead(e.target.value)}
                className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm bg-white"
              >
                {branches.map((branch) => (
                  <option key={branch.name} value={branch.name}>
                    {branch.name}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {branchError && (
            <p className="mb-4 text-sm text-[#d1242f]">{branchError}</p>
          )}

          <div className="mb-4">
            <label htmlFor="title" className="block text-sm font-semibold mb-1">
              Title <span className="text-[#d1242f]">*</span>
            </label>
            <input
              id="title"
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#0969da]"
              placeholder="Pull request title"
              maxLength={256}
            />
          </div>

          <div className="mb-4">
            <div className="flex justify-between items-center mb-1">
              <label htmlFor="body" className="text-sm font-semibold">
                Body
              </label>
              <button
                type="button"
                onClick={() => setShowPreview((p) => !p)}
                className="text-sm text-[#0969da] hover:underline"
              >
                {showPreview ? "Edit" : "Preview"}
              </button>
            </div>
            {showPreview ? (
              <div
                className="min-h-[200px] p-3 border border-[#d0d7de] rounded-md prose prose-sm max-w-none"
                dangerouslySetInnerHTML={{
                  __html: body.trim()
                    ? renderMarkdown(body)
                    : "<p class='text-[#656d76]'>Nothing to preview</p>",
                }}
              />
            ) : (
              <textarea
                id="body"
                value={body}
                onChange={(e) => setBody(e.target.value)}
                className="w-full min-h-[200px] px-3 py-2 border border-[#d0d7de] rounded-md text-sm resize-y focus:outline-none focus:ring-2 focus:ring-[#0969da]"
                placeholder="Describe your changes in GitHub Flavored Markdown"
              />
            )}
          </div>

          {error && <p className="mb-4 text-sm text-[#d1242f]">{error}</p>}

          <div className="flex gap-2 justify-end">
            <Link
              href={`/${owner}/${repo}/pulls`}
              className="px-4 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white hover:bg-[#f6f8fa]"
            >
              Cancel
            </Link>
            <button
              type="submit"
              disabled={!title.trim() || !!branchError || submitting}
              className="px-4 py-1.5 text-sm bg-[#1f883d] text-white rounded-md font-semibold border border-black/10 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {submitting ? "Creating…" : "Create pull request"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
