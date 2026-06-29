"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import CommentForm from "@/components/issue/CommentForm";
import { useAuth } from "@/components/providers/auth-provider";
import { apiClient, isApiError } from "@/lib/api-client";
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
type CurrentUser = { login: string };
type OrgMember = { login: string; role: string };

type Props = {
  params: Promise<{ owner: string; repo: string; number: string }>;
};

const WRITE_ROLES = new Set(["write", "admin", "maintainer", "owner"]);

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
  const { token } = useAuth();
  const [owner, setOwner] = useState("");
  const [repo, setRepo] = useState("");
  const [number, setNumber] = useState("");
  const [issue, setIssue] = useState<Issue | null>(null);
  const [optimisticState, setOptimisticState] = useState<"open" | "closed">("open");
  const [comments, setComments] = useState<Comment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [togglingState, setTogglingState] = useState(false);
  const [canManageIssue, setCanManageIssue] = useState(false);
  const [toastMessage, setToastMessage] = useState<{
    message: string;
    variant: "success" | "error";
  } | null>(null);

  const toast = useMemo(
    () => ({
      success(message: string) {
        setToastMessage({ message, variant: "success" });
      },
      error(message: string) {
        setToastMessage({ message, variant: "error" });
      },
    }),
    [],
  );

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
      const loadedIssue = (await issueRes.json()) as Issue;
      setIssue(loadedIssue);
      setOptimisticState(loadedIssue.state);
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

  useEffect(() => {
    if (!issue || !token) {
      setCanManageIssue(false);
      return;
    }

    const authToken = token;
    const currentIssue = issue;
    let cancelled = false;

    async function checkPermissions() {
      try {
        const currentUser = await apiClient.get<CurrentUser>("/api/v3/user", {
          token: authToken,
        });
        if (cancelled) return;

        if (currentUser.login === currentIssue.user.login) {
          setCanManageIssue(true);
          return;
        }

        if (currentUser.login === owner) {
          setCanManageIssue(true);
          return;
        }

        try {
          const members = await apiClient.get<OrgMember[]>(
            `/api/v3/orgs/${owner}/members`,
            { token: authToken },
          );
          if (cancelled) return;
          const membership = members.find((m) => m.login === currentUser.login);
          setCanManageIssue(
            membership != null && WRITE_ROLES.has(membership.role),
          );
        } catch {
          if (!cancelled) setCanManageIssue(false);
        }
      } catch {
        if (!cancelled) setCanManageIssue(false);
      }
    }

    void checkPermissions();

    return () => {
      cancelled = true;
    };
  }, [issue, owner, token]);

  useEffect(() => {
    if (!toastMessage) return;
    const timer = window.setTimeout(() => setToastMessage(null), 4000);
    return () => window.clearTimeout(timer);
  }, [toastMessage]);

  const toggleState = async () => {
    if (!issue || !token) return;
    const nextState = optimisticState === "open" ? "closed" : "open";
    const previousState = optimisticState;
    setOptimisticState(nextState);
    setTogglingState(true);
    try {
      await apiClient.patch(
        `/api/v3/repos/${owner}/${repo}/issues/${number}`,
        { state: nextState },
        { token },
      );
      setIssue((prev) => (prev ? { ...prev, state: nextState } : prev));
      toast.success(nextState === "closed" ? "Issue closed" : "Issue reopened");
    } catch (e) {
      setOptimisticState(previousState);
      if (isApiError(e)) {
        if (e.status === 403) {
          toast.error("Permission denied");
        } else if (e.status === 409) {
          if (
            window.confirm(
              "This issue was updated elsewhere. Reload the page to see the latest state?",
            )
          ) {
            window.location.reload();
          }
        } else {
          setError(e.message);
        }
      } else {
        setError(e instanceof Error ? e.message : "Failed to update issue state");
      }
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
          {canManageIssue && (
            <button
              type="button"
              onClick={toggleState}
              disabled={togglingState}
              className={`px-4 py-1.5 text-sm rounded-md font-semibold border disabled:opacity-50 ${
                optimisticState === "open"
                  ? "bg-[#8250df] text-white border-black/10 hover:bg-[#6f42c1]"
                  : "bg-[#1f883d] text-white border-black/10 hover:bg-[#1a7f37]"
              }`}
            >
              {togglingState
                ? "Updating…"
                : optimisticState === "open"
                  ? "Close issue"
                  : "Reopen issue"}
            </button>
          )}
        </div>

        <div className="mb-4 flex items-center gap-2 flex-wrap">
          <span
            className={`inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-semibold text-white ${
              optimisticState === "open" ? "bg-[#1a7f37]" : "bg-[#8250df]"
            }`}
          >
            {optimisticState === "open" ? "⊙ Open" : "✓ Closed"}
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

      {toastMessage && (
        <div
          role="status"
          className={`fixed bottom-6 right-6 px-4 py-2 rounded-md text-sm font-medium shadow-lg ${
            toastMessage.variant === "success"
              ? "bg-[#1f883d] text-white"
              : "bg-[#d1242f] text-white"
          }`}
        >
          {toastMessage.message}
        </div>
      )}
    </div>
  );
}
