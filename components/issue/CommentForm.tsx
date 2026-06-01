"use client";

import { useState } from "react";
import { renderMarkdown } from "@/lib/markdown";

type CommentFormProps = {
  owner: string;
  repo: string;
  issueNumber: number;
  onSubmitted?: () => void;
};

export default function CommentForm({ owner, repo, issueNumber, onSubmitted }: CommentFormProps) {
  const [tab, setTab] = useState<"write" | "preview">("write");
  const [body, setBody] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!body.trim()) return;

    setSubmitting(true);
    setError(null);
    try {
      const res = await fetch(`/repos/${owner}/${repo}/issues/${issueNumber}/comments`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ body }),
      });
      if (!res.ok) throw new Error("Failed to post comment");
      setBody("");
      setTab("write");
      onSubmitted?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to post comment");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="mt-4">
      <div className="flex gap-0 border border-[#d0d7de] border-b-0 rounded-t-md bg-[#f6f8fa]">
        <button
          type="button"
          onClick={() => setTab("write")}
          className={`px-4 py-2 text-sm border-b-2 ${
            tab === "write"
              ? "border-[#fd8c73] font-semibold bg-white"
              : "border-transparent text-[#656d76] hover:text-[#1f2328]"
          }`}
        >
          Write
        </button>
        <button
          type="button"
          onClick={() => setTab("preview")}
          className={`px-4 py-2 text-sm border-b-2 ${
            tab === "preview"
              ? "border-[#fd8c73] font-semibold bg-white"
              : "border-transparent text-[#656d76] hover:text-[#1f2328]"
          }`}
        >
          Preview
        </button>
      </div>

      {tab === "write" ? (
        <textarea
          className="w-full min-h-[120px] p-3 border border-[#d0d7de] rounded-b-md text-sm resize-y focus:outline-none focus:ring-2 focus:ring-[#0969da]"
          placeholder="Leave a comment"
          value={body}
          onChange={(e) => setBody(e.target.value)}
        />
      ) : (
        <div
          className="min-h-[120px] p-3 border border-[#d0d7de] rounded-b-md text-sm prose prose-sm max-w-none bg-white"
          dangerouslySetInnerHTML={{ __html: body.trim() ? renderMarkdown(body) : "<p class='text-[#656d76]'>Nothing to preview</p>" }}
        />
      )}

      {error && <p className="mt-2 text-sm text-[#d1242f]">{error}</p>}

      <div className="flex justify-end mt-3">
        <button
          type="submit"
          disabled={!body.trim() || submitting}
          className="px-4 py-1.5 text-sm bg-[#1f883d] text-white rounded-md font-semibold border border-black/10 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {submitting ? "Submitting…" : "Comment"}
        </button>
      </div>
    </form>
  );
}
