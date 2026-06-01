"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import CommentForm from "@/components/issue/CommentForm";
import { renderMarkdown } from "@/lib/markdown";

type Label = { name: string; color: string };
type User = { login: string };
type Issue = {
  number: number;
  title: string;
  body: string;
  state: "open" | "closed";
  user: User;
  labels: Label[];
  created_at: string;
  comments: number;
};
type Comment = {
  id: number;
  body: string;
  user: User;
  created_at: string;
};

type Props = {
  params: Promise<{ owner: string; repo: string; number: string }>;
};

function formatAge(dateStr: string): string {
  const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} minute${minutes === 1 ? "" : "s"} ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} hour${hours === 1 ? "" : "s"} ago`;
  const days = Math.floor(hours / 24);
  return `${days} day${days === 1 ? "" : "s"} ago`;
}

function labelStyle(color: string): React.CSSProperties {
  const hex = color.startsWith("#") ? color : `#${color}`;
  return { backgroundColor: hex, color: "#fff" };
}

export default function IssueDetailPage({ params }: Props) {
  const [owner, setOwner] = useState("");
  const [repo, setRepo] = useState("");
  const [number, setNumber] = useState("");
  const [issue, setIssue] = useState<Issue | null>(null);
  const [comments, setComments] = useState<Comment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [togglingState, setTogglingState] = useState(false);

  useEffect(() => {
    params.then(({ owner: o, repo: r, number: n }) => {
      setOwner(o);
      setRepo(r);
      setNumber(n);
    });
  }, [params]);

  const loadIssue = useCallback(async () => {
    if (!owner || !repo || !number) return;
    setLoading(true);
    setError(null);
    try {
      const [issueRes, commentsRes] = await Promise.all([
        fetch(`/repos/${owner}/${repo}/issues/${number}`),
        fetch(`/repos/${owner}/${repo}/issues/${number}/comments`),
      ]);
      if (!issueRes.ok) throw new Error("Issue not found");
      setIssue((await issueRes.json()) as Issue);
      if (commentsRes.ok) setComments((await commentsRes.json()) as Comment[]);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load issue");
    } finally {
      setLoading(false);
    }
  }, [owner, repo, number]);

  useEffect(() => {
    loadIssue();
  }, [loadIssue]);

  const toggleState = async () => {
    if (!issue) return;
    setTogglingState(true);
    try {
      const nextState = issue.state === "open" ? "closed" : "open";
      const res = await fetch(`/repos/${owner}/${repo}/issues/${number}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ state: nextState }),
      });
      if (!res.ok) throw new Error("Failed to update issue state");
      setIssue((await res.json()) as Issue);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to update issue state");
    } finally {
      setTogglingState(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-[#f6f8fa] flex items-center justify-center text-[#656d76]">
        Loading issue…
      </div>
    );
  }

  if (error || !issue) {
    return (
      <div className="min-h-screen bg-[#f6f8fa] flex items-center justify-center">
        <div className="text-center">
          <p className="text-[#d1242f] mb-4">{error ?? "Issue not found"}</p>
          <Link href={`/${owner}/${repo}/issues`} className="text-[#0969da] hover:underline">
            ← Back to issues
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <div className="max-w-[960px] mx-auto px-6 py-6">
        <div className="mb-4">
          <Link href={`/${owner}/${repo}/issues`} className="text-sm text-[#0969da] hover:underline">
            ← Back to issues
          </Link>
        </div>

        <div className="flex justify-between items-start gap-4 mb-4">
          <h1 className="text-2xl font-semibold">
            {issue.title}{" "}
            <span className="text-[#656d76] font-normal">#{issue.number}</span>
          </h1>
          <button
            type="button"
            onClick={toggleState}
            disabled={togglingState}
            className={`px-4 py-1.5 text-sm rounded-md font-semibold border disabled:opacity-50 ${
              issue.state === "open"
                ? "bg-[#8250df] text-white border-black/10 hover:bg-[#6f42c1]"
                : "bg-[#1f883d] text-white border-black/10 hover:bg-[#1a7f37]"
            }`}
          >
            {togglingState ? "Updating…" : issue.state === "open" ? "Close issue" : "Reopen issue"}
          </button>
        </div>

        <div className="mb-4 flex items-center gap-2 flex-wrap">
          <span
            className={`inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-semibold text-white ${
              issue.state === "open" ? "bg-[#1a7f37]" : "bg-[#8250df]"
            }`}
          >
            {issue.state === "open" ? "⊙ Open" : "✓ Closed"}
          </span>
          {issue.labels.map((label) => (
            <span
              key={label.name}
              className="px-2 py-0.5 rounded-full text-[11px] font-semibold"
              style={labelStyle(label.color)}
            >
              {label.name}
            </span>
          ))}
          <span className="text-sm text-[#656d76]">
            <strong>{issue.user.login}</strong> opened this issue {formatAge(issue.created_at)} ·{" "}
            {issue.comments} comment{issue.comments === 1 ? "" : "s"}
          </span>
        </div>

        <div className="bg-white border border-[#d0d7de] rounded-md p-4 mb-4">
          <div className="flex items-center gap-2 mb-3">
            <span className="w-8 h-8 rounded-full bg-gradient-to-br from-[#0969da] to-[#8250df] text-white text-xs font-semibold inline-flex items-center justify-center">
              {issue.user.login.slice(0, 2).toUpperCase()}
            </span>
            <span className="font-semibold text-sm">{issue.user.login}</span>
            <span className="text-xs text-[#656d76]">commented {formatAge(issue.created_at)}</span>
          </div>
          <div
            className="prose prose-sm max-w-none pl-10"
            dangerouslySetInnerHTML={{ __html: renderMarkdown(issue.body ?? "") }}
          />
        </div>

        <div className="space-y-4 mb-4">
          {comments.map((comment) => (
            <div key={comment.id} className="bg-white border border-[#d0d7de] rounded-md p-4">
              <div className="flex items-center gap-2 mb-3">
                <span className="w-8 h-8 rounded-full bg-gradient-to-br from-[#0969da] to-[#8250df] text-white text-xs font-semibold inline-flex items-center justify-center">
                  {comment.user.login.slice(0, 2).toUpperCase()}
                </span>
                <span className="font-semibold text-sm">{comment.user.login}</span>
                <span className="text-xs text-[#656d76]">commented {formatAge(comment.created_at)}</span>
              </div>
              <div
                className="prose prose-sm max-w-none pl-10"
                dangerouslySetInnerHTML={{ __html: renderMarkdown(comment.body) }}
              />
            </div>
          ))}
        </div>

        <CommentForm
          owner={owner}
          repo={repo}
          issueNumber={issue.number}
          onSubmitted={loadIssue}
        />
      </div>
    </div>
  );
}
