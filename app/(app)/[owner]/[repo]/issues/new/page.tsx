"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { renderMarkdown } from "@/lib/markdown";

type Label = { name: string; color: string };

type Props = {
  params: Promise<{ owner: string; repo: string }>;
};

export default function NewIssuePage({ params }: Props) {
  const router = useRouter();
  const [owner, setOwner] = useState("");
  const [repo, setRepo] = useState("");
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [showPreview, setShowPreview] = useState(false);
  const [labels, setLabels] = useState<Label[]>([]);
  const [selectedLabels, setSelectedLabels] = useState<string[]>([]);
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
    fetch(`/repos/${owner}/${repo}/labels?per_page=100`)
      .then((res) => (res.ok ? res.json() : []))
      .then((data: Label[]) => setLabels(data))
      .catch(() => setLabels([]));
  }, [owner, repo]);

  const titleError =
    title.length === 0 ? null : title.length > 256 ? "Title must be 256 characters or fewer" : null;

  const toggleLabel = (name: string) => {
    setSelectedLabels((prev) =>
      prev.includes(name) ? prev.filter((l) => l !== name) : [...prev, name],
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim()) {
      setError("Title is required");
      return;
    }
    if (title.length > 256) {
      setError("Title must be 256 characters or fewer");
      return;
    }

    setSubmitting(true);
    setError(null);
    try {
      const res = await fetch(`/repos/${owner}/${repo}/issues`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          title: title.trim(),
          body,
          labels: selectedLabels,
        }),
      });
      if (!res.ok) throw new Error("Failed to create issue");
      const issue = (await res.json()) as { number: number };
      router.push(`/${owner}/${repo}/issues/${issue.number}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create issue");
      setSubmitting(false);
    }
  };

  if (!owner || !repo) return null;

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <div className="max-w-[960px] mx-auto px-6 py-6">
        <div className="mb-4">
          <Link href={`/${owner}/${repo}/issues`} className="text-sm text-[#0969da] hover:underline">
            ← Back to issues
          </Link>
        </div>

        <h1 className="text-2xl font-semibold mb-6">
          New Issue in{" "}
          <span className="text-[#0969da]">
            {owner}/{repo}
          </span>
        </h1>

        <form onSubmit={handleSubmit} className="bg-white border border-[#d0d7de] rounded-md p-6">
          <div className="mb-4">
            <label htmlFor="title" className="block text-sm font-semibold mb-1">
              Title <span className="text-[#d1242f]">*</span>
            </label>
            <input
              id="title"
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className={`w-full px-3 py-2 border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#0969da] ${
                titleError ? "border-[#d1242f]" : "border-[#d0d7de]"
              }`}
              placeholder="Issue title"
              maxLength={256}
            />
            <div className="flex justify-between mt-1">
              {titleError ? (
                <span className="text-xs text-[#d1242f]">{titleError}</span>
              ) : (
                <span className="text-xs text-[#656d76]">Required</span>
              )}
              <span className={`text-xs ${title.length > 256 ? "text-[#d1242f]" : "text-[#656d76]"}`}>
                {title.length}/256
              </span>
            </div>
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
                  __html: body.trim() ? renderMarkdown(body) : "<p class='text-[#656d76]'>Nothing to preview</p>",
                }}
              />
            ) : (
              <textarea
                id="body"
                value={body}
                onChange={(e) => setBody(e.target.value)}
                className="w-full min-h-[200px] px-3 py-2 border border-[#d0d7de] rounded-md text-sm resize-y focus:outline-none focus:ring-2 focus:ring-[#0969da]"
                placeholder="Write your issue description in GitHub Flavored Markdown"
              />
            )}
          </div>

          {labels.length > 0 && (
            <div className="mb-4">
              <span className="block text-sm font-semibold mb-2">Labels</span>
              <div className="flex flex-wrap gap-3">
                {labels.map((label) => (
                  <label key={label.name} className="flex items-center gap-1.5 text-sm cursor-pointer">
                    <input
                      type="checkbox"
                      checked={selectedLabels.includes(label.name)}
                      onChange={() => toggleLabel(label.name)}
                    />
                    <span
                      className="px-2 py-0.5 rounded-full text-[11px] font-semibold text-white"
                      style={{ backgroundColor: label.color.startsWith("#") ? label.color : `#${label.color}` }}
                    >
                      {label.name}
                    </span>
                  </label>
                ))}
              </div>
            </div>
          )}

          {error && <p className="mb-4 text-sm text-[#d1242f]">{error}</p>}

          <div className="flex gap-2 justify-end">
            <Link
              href={`/${owner}/${repo}/issues`}
              className="px-4 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white hover:bg-[#f6f8fa]"
            >
              Cancel
            </Link>
            <button
              type="submit"
              disabled={!title.trim() || !!titleError || submitting}
              className="px-4 py-1.5 text-sm bg-[#1f883d] text-white rounded-md font-semibold border border-black/10 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {submitting ? "Creating…" : "Submit new issue"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
