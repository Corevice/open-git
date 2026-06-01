"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import CommentForm from "@/components/issue/CommentForm";
import MergePanel from "@/components/pr/MergePanel";
import { ApiError } from "@/lib/api";
import type { PullRequest } from "@/lib/api-types";
import { renderMarkdown } from "@/lib/markdown";

type User = { login: string };
type ReviewComment = {
  id: number;
  body: string;
  user: User;
  created_at: string;
  path?: string;
  line?: number;
};
type Commit = {
  sha: string;
  commit: {
    message: string;
    author: { name: string; date: string };
  };
};
type FileDiff = {
  filename: string;
  patch?: string;
  additions: number;
  deletions: number;
  status?: string;
};
type PRDetail = {
  id: number;
  number: number;
  title: string;
  body: string;
  state: string;
  draft?: boolean;
  merged_at?: string | null;
  mergeable_state?: string;
  user: User;
  created_at: string;
  head: { ref: string };
  base: { ref: string };
  commits?: number;
};

type Props = {
  params: Promise<{ owner: string; repo: string; number: string }>;
};

type Tab = "conversation" | "commits" | "files";

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

function toPullRequest(pr: PRDetail): PullRequest {
  const hasConflicts =
    pr.mergeable_state === "dirty" ||
    pr.mergeable_state === "conflicting" ||
    pr.state === "conflict" ||
    pr.state === "conflicts";

  return {
    id: pr.id,
    number: pr.number,
    headRef: pr.head.ref,
    baseRef: pr.base.ref,
    state: hasConflicts ? "conflicts" : pr.merged_at ? "merged" : pr.state,
    mergedAt: pr.merged_at ?? null,
  };
}

function prStateBadge(pr: PRDetail): { label: string; className: string } {
  if (pr.merged_at) {
    return { label: "Merged", className: "bg-[#8250df] text-white" };
  }
  if (pr.draft) {
    return { label: "Draft", className: "bg-[#eaeef2] text-[#57606a]" };
  }
  if (
    pr.mergeable_state === "dirty" ||
    pr.mergeable_state === "conflicting" ||
    pr.state === "conflict" ||
    pr.state === "conflicts"
  ) {
    return { label: "Conflicts", className: "bg-[#ffebe9] text-[#cf222e]" };
  }
  if (pr.state === "open") {
    return { label: "Open", className: "bg-[#1a7f37] text-white" };
  }
  return { label: "Closed", className: "bg-[#8250df] text-white" };
}

function DiffViewer({ file }: { file: FileDiff }) {
  const lines = file.patch?.split("\n") ?? [];

  return (
    <div className="border border-[#d0d7de] rounded-lg mb-4 overflow-hidden">
      <div className="bg-[#f6f8fa] px-4 py-2.5 border-b border-[#d0d7de] font-mono text-[13px] flex justify-between">
        <span>{file.filename}</span>
        <span>
          <span className="text-[#1a7f37]">+{file.additions}</span>{" "}
          <span className="text-[#cf222e]">-{file.deletions}</span>
        </span>
      </div>
      {lines.length === 0 ? (
        <div className="px-4 py-3 text-sm text-[#656d76]">Binary file or no diff available.</div>
      ) : (
        <table className="w-full border-collapse font-mono text-xs">
          <tbody>
            {lines.map((line, i) => {
              if (line.startsWith("@@")) {
                return (
                  <tr key={i} className="bg-[#ddf4ff] text-[#57606a]">
                    <td colSpan={2} className="px-2.5 py-1">
                      {line}
                    </td>
                  </tr>
                );
              }
              if (line.startsWith("+") && !line.startsWith("+++")) {
                return (
                  <tr key={i} className="bg-[#dafbe1]">
                    <td className="w-10 text-right text-[#6e7781] bg-[#ccffd8] px-2.5 select-none" />
                    <td className="px-2.5">{line}</td>
                  </tr>
                );
              }
              if (line.startsWith("-") && !line.startsWith("---")) {
                return (
                  <tr key={i} className="bg-[#ffebe9]">
                    <td className="w-10 text-right text-[#6e7781] bg-[#ffd7d5] px-2.5 select-none" />
                    <td className="px-2.5">{line}</td>
                  </tr>
                );
              }
              return (
                <tr key={i}>
                  <td className="w-10 text-right text-[#6e7781] bg-[#f6f8fa] px-2.5 select-none" />
                  <td className="px-2.5">{line || " "}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}
    </div>
  );
}

export default function PullRequestDetailPage({ params }: Props) {
  const [owner, setOwner] = useState("");
  const [repo, setRepo] = useState("");
  const [number, setNumber] = useState("");
  const [pr, setPr] = useState<PRDetail | null>(null);
  const [commits, setCommits] = useState<Commit[]>([]);
  const [reviewComments, setReviewComments] = useState<ReviewComment[]>([]);
  const [files, setFiles] = useState<FileDiff[]>([]);
  const [tab, setTab] = useState<Tab>("conversation");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    params.then(({ owner: o, repo: r, number: n }) => {
      setOwner(o);
      setRepo(r);
      setNumber(n);
    });
  }, [params]);

  const loadPullRequest = useCallback(async () => {
    if (!owner || !repo || !number) return;
    setLoading(true);
    setError(null);
    try {
      const [prRes, commitsRes, commentsRes, filesRes] = await Promise.all([
        fetch(`/repos/${owner}/${repo}/pulls/${number}`),
        fetch(`/repos/${owner}/${repo}/pulls/${number}/commits`),
        fetch(`/repos/${owner}/${repo}/pulls/${number}/comments`),
        fetch(`/repos/${owner}/${repo}/pulls/${number}/files`),
      ]);

      if (!prRes.ok) throw new Error("Pull request not found");

      setPr((await prRes.json()) as PRDetail);
      if (commitsRes.ok) setCommits((await commitsRes.json()) as Commit[]);
      if (commentsRes.ok) setReviewComments((await commentsRes.json()) as ReviewComment[]);
      if (filesRes.ok) setFiles((await filesRes.json()) as FileDiff[]);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load pull request");
    } finally {
      setLoading(false);
    }
  }, [owner, repo, number]);

  useEffect(() => {
    loadPullRequest();
  }, [loadPullRequest]);

  const handleMerge = async (method: string) => {
    const res = await fetch(`/repos/${owner}/${repo}/pulls/${number}/merge`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ merge_method: method }),
    });

    if (!res.ok) {
      let message = res.statusText;
      try {
        const body = (await res.json()) as { message?: string };
        message = body.message ?? message;
      } catch {
        // ignore JSON parse errors
      }
      throw new ApiError(res.status, message);
    }

    await loadPullRequest();
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-[#f6f8fa] flex items-center justify-center text-[#656d76]">
        Loading pull request…
      </div>
    );
  }

  if (error || !pr) {
    return (
      <div className="min-h-screen bg-[#f6f8fa] flex items-center justify-center">
        <div className="text-center">
          <p className="text-[#d1242f] mb-4">{error ?? "Pull request not found"}</p>
          <Link href={`/${owner}/${repo}/pulls`} className="text-[#0969da] hover:underline">
            ← Back to pull requests
          </Link>
        </div>
      </div>
    );
  }

  const badge = prStateBadge(pr);

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <div className="max-w-[1280px] mx-auto px-6 py-6">
        <div className="mb-4">
          <Link href={`/${owner}/${repo}/pulls`} className="text-sm text-[#0969da] hover:underline">
            ← Back to pull requests
          </Link>
        </div>

        <div className="bg-white border border-[#d0d7de] rounded-lg p-5 mb-4">
          <h1 className="text-2xl font-semibold mb-3">
            {pr.title}{" "}
            <span className="text-[#656d76] font-normal">#{pr.number}</span>
          </h1>
          <div className="flex items-center gap-2 flex-wrap text-sm text-[#656d76]">
            <span
              className={`inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-semibold ${badge.className}`}
            >
              {badge.label}
            </span>
            <span>
              <strong>{pr.user.login}</strong> wants to merge{" "}
              <strong>{commits.length || pr.commits || 0} commits</strong> into{" "}
              <code className="bg-[#f6f8fa] px-1.5 py-0.5 rounded text-xs">{pr.base.ref}</code> from{" "}
              <code className="bg-[#f6f8fa] px-1.5 py-0.5 rounded text-xs">{pr.head.ref}</code>
            </span>
            <span>· opened {formatAge(pr.created_at)}</span>
          </div>
        </div>

        <div className="flex gap-0 bg-white border border-b-0 border-[#d0d7de] rounded-t-lg px-4">
          {(
            [
              { id: "conversation" as const, label: "Conversation", count: reviewComments.length + 1 },
              { id: "commits" as const, label: "Commits", count: commits.length },
              { id: "files" as const, label: "Files Changed", count: files.length },
            ] as const
          ).map(({ id, label, count }) => (
            <button
              key={id}
              type="button"
              onClick={() => setTab(id)}
              className={`px-4 py-3.5 text-sm border-b-2 inline-flex items-center gap-2 ${
                tab === id
                  ? "border-[#fd8c73] font-semibold text-[#1f2328]"
                  : "border-transparent text-[#656d76] hover:text-[#0969da]"
              }`}
            >
              {label}
              <span className="text-xs bg-[#eaeef2] px-1.5 py-0.5 rounded-full">{count}</span>
            </button>
          ))}
        </div>

        <div className="bg-white border border-[#d0d7de] rounded-b-lg p-5">
          {tab === "conversation" && (
            <>
              <div className="bg-white border border-[#d0d7de] rounded-md p-4 mb-4">
                <div className="flex items-center gap-2 mb-3">
                  <span className="w-8 h-8 rounded-full bg-gradient-to-br from-[#0969da] to-[#8250df] text-white text-xs font-semibold inline-flex items-center justify-center">
                    {pr.user.login.slice(0, 2).toUpperCase()}
                  </span>
                  <span className="font-semibold text-sm">{pr.user.login}</span>
                  <span className="text-xs text-[#656d76]">commented {formatAge(pr.created_at)}</span>
                </div>
                <div
                  className="prose prose-sm max-w-none pl-10"
                  dangerouslySetInnerHTML={{ __html: renderMarkdown(pr.body ?? "") }}
                />
              </div>

              {reviewComments.map((comment) => (
                <div key={comment.id} className="bg-white border border-[#d0d7de] rounded-md p-4 mb-4">
                  <div className="flex items-center gap-2 mb-3">
                    <span className="w-8 h-8 rounded-full bg-gradient-to-br from-[#0969da] to-[#8250df] text-white text-xs font-semibold inline-flex items-center justify-center">
                      {comment.user.login.slice(0, 2).toUpperCase()}
                    </span>
                    <span className="font-semibold text-sm">{comment.user.login}</span>
                    <span className="text-xs text-[#656d76]">
                      reviewed {formatAge(comment.created_at)}
                      {comment.path ? ` on ${comment.path}${comment.line ? `:${comment.line}` : ""}` : ""}
                    </span>
                  </div>
                  <div
                    className="prose prose-sm max-w-none pl-10"
                    dangerouslySetInnerHTML={{ __html: renderMarkdown(comment.body) }}
                  />
                </div>
              ))}

              <CommentForm
                owner={owner}
                repo={repo}
                issueNumber={pr.number}
                onSubmitted={loadPullRequest}
              />

              <MergePanel pr={toPullRequest(pr)} onMerge={handleMerge} />
            </>
          )}

          {tab === "commits" && (
            <div className="divide-y divide-[#d0d7de]">
              {commits.length === 0 ? (
                <p className="text-[#656d76] py-4">No commits found.</p>
              ) : (
                commits.map((commit) => (
                  <div key={commit.sha} className="py-3 flex items-start gap-3">
                    <span className="text-[#8250df] mt-0.5">●</span>
                    <div>
                      <p className="font-semibold text-sm">
                        {commit.commit.message.split("\n")[0]}
                      </p>
                      <p className="text-xs text-[#656d76] mt-1">
                        {commit.commit.author.name} committed {formatAge(commit.commit.author.date)} ·{" "}
                        <code className="bg-[#f6f8fa] px-1 rounded">{commit.sha.slice(0, 7)}</code>
                      </p>
                    </div>
                  </div>
                ))
              )}
            </div>
          )}

          {tab === "files" && (
            <div>
              {files.length === 0 ? (
                <p className="text-[#656d76] py-4">No file changes found.</p>
              ) : (
                files.map((file) => <DiffViewer key={file.filename} file={file} />)
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
