"use client";

import { useState } from "react";

type FeedbackWidgetProps = {
  path: string;
  version?: string;
};

export function FeedbackWidget({
  path,
  version = "latest",
}: FeedbackWidgetProps) {
  const [submitted, setSubmitted] = useState(false);
  const [pending, setPending] = useState(false);

  async function submitFeedback(helpful: boolean) {
    if (pending || submitted) {
      return;
    }

    setPending(true);

    try {
      const response = await fetch("/api/docs/feedback", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path, helpful, version }),
      });

      if (response.ok) {
        setSubmitted(true);
      }
    } catch {
      // Fail silently when the feedback endpoint is unavailable.
    } finally {
      setPending(false);
    }
  }

  if (submitted) {
    return (
      <div
        aria-label="ドキュメントフィードバック"
        style={{ marginTop: "32px", paddingTop: "16px" }}
      >
        <p style={{ color: "#166534", margin: 0 }}>
          ごフィードバックありがとうございます。
        </p>
      </div>
    );
  }

  return (
    <div
      aria-label="ドキュメントフィードバック"
      style={{
        borderTop: "1px solid #e5e7eb",
        display: "flex",
        flexWrap: "wrap",
        gap: "8px",
        marginTop: "32px",
        paddingTop: "16px",
      }}
    >
      <span style={{ marginRight: "8px" }}>このページは役に立ちましたか？</span>
      <button
        type="button"
        disabled={pending}
        onClick={() => submitFeedback(true)}
        style={{
          backgroundColor: "#16a34a",
          border: "none",
          borderRadius: "6px",
          color: "#ffffff",
          cursor: pending ? "not-allowed" : "pointer",
          opacity: pending ? 0.7 : 1,
          padding: "6px 12px",
        }}
      >
        役に立った
      </button>
      <button
        type="button"
        disabled={pending}
        onClick={() => submitFeedback(false)}
        style={{
          backgroundColor: "#ffffff",
          border: "1px solid #d1d5db",
          borderRadius: "6px",
          color: "#374151",
          cursor: pending ? "not-allowed" : "pointer",
          opacity: pending ? 0.7 : 1,
          padding: "6px 12px",
        }}
      >
        役に立たなかった
      </button>
    </div>
  );
}
