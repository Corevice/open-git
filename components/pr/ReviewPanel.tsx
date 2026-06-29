"use client";

import { useState } from "react";

export interface ReviewPanelProps {
  owner: string;
  repo: string;
  prNumber: number;
  onSubmitted?: () => void;
}

type ReviewEvent = "COMMENT" | "APPROVE" | "CHANGES_REQUESTED";

const REVIEW_EVENTS: { value: ReviewEvent; label: string }[] = [
  { value: "COMMENT", label: "Comment" },
  { value: "APPROVE", label: "Approve" },
  { value: "CHANGES_REQUESTED", label: "Request changes" },
];

export default function ReviewPanel({
  owner,
  repo,
  prNumber,
  onSubmitted,
}: ReviewPanelProps) {
  const [event, setEvent] = useState<ReviewEvent>("COMMENT");
  const [body, setBody] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);

    try {
      const res = await fetch(`/repos/${owner}/${repo}/pulls/${prNumber}/reviews`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ event, body }),
      });

      if (res.ok) {
        setEvent("COMMENT");
        setBody("");
        onSubmitted?.();
        return;
      }

      let message = res.statusText;
      try {
        const data = (await res.json()) as { message?: string };
        message = data.message ?? message;
      } catch {
        // ignore JSON parse errors
      }
      setError(message);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to submit review");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="border border-[#d0d7de] rounded-lg p-4 mt-4 bg-white">
      <h3 className="text-sm font-semibold mb-3">Submit review</h3>

      <label htmlFor="review-event" className="block text-sm text-[#656d76] mb-1">
        Review type
      </label>
      <select
        id="review-event"
        value={event}
        onChange={(e) => setEvent(e.target.value as ReviewEvent)}
        disabled={submitting}
        className="w-full mb-3 px-3 py-2 text-sm border border-[#d0d7de] rounded-md bg-white disabled:opacity-50"
      >
        {REVIEW_EVENTS.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>

      <label htmlFor="review-body" className="block text-sm text-[#656d76] mb-1">
        Review comment
      </label>
      <textarea
        id="review-body"
        className="w-full min-h-[100px] p-3 border border-[#d0d7de] rounded-md text-sm resize-y focus:outline-none focus:ring-2 focus:ring-[#0969da] disabled:opacity-50"
        placeholder="Leave a review comment (optional)"
        value={body}
        onChange={(e) => setBody(e.target.value)}
        disabled={submitting}
      />

      {error && (
        <p className="mt-3 text-sm text-[#cf222e] bg-[#ffebe9] border border-[#ff8182] rounded-md px-3 py-2">
          {error}
        </p>
      )}

      <div className="mt-3">
        <button
          type="submit"
          disabled={submitting}
          className="px-4 py-2 bg-[#1f883d] text-white rounded-md text-sm font-semibold border border-black/10 hover:bg-[#1a7f37] disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {submitting ? "Submitting…" : "Submit review"}
        </button>
      </div>
    </form>
  );
}
