"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

import CommentForm from "@/components/issue/CommentForm";
import MergePanel from "@/components/pr/MergePanel";
import DiffViewer from "@/components/repo/DiffViewer";
import {
  ApiError,
  getPullRequest,
  getPullRequestFiles,
  listReviewComments,
  listReviews,
  mergePullRequest,
} from "@/lib/api";
import type { PullRequest as MergePanelPullRequest } from "@/lib/api-types";
import { renderMarkdown } from "@/lib/markdown";
import type { PullRequest, PullRequestFile, Review, ReviewComment } from "@/types/pull-request";

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

function authorLogin(pr: PullRequest): string {
  return pr.user?.login ?? pr.author_id ?? "unknown";
}

function toMergePanelPullRequest(pr: PullRequest): MergePanelPullRequest {
  const hasConflicts =
    pr.mergeable === false ||
    pr.mergeable_state === "dirty" ||
    pr.mergeable_state === "conflicting";

  return {
    id: Number.isNaN(Number(pr.id)) ? 0 : Number(pr.id),
    number: pr.number,
    headRef: pr.head_ref,
    baseRef: pr.base_ref,
    state: hasConflicts ? "conflicts" : pr.merged_at ? "merged" : pr.state,
    mergedAt: pr.merged_at,
  };
}

function prStateBadge(pr: PullRequest): { label: string; className: string } {
  if (pr.merged_at) {
    return { label: "Merged", className: "bg-[#8250df] text-white" };
  }
  if (pr.draft) {
    return { label: "Draft", className: "bg-[#eaeef2] text-[#57606a]" };
  }
  if (
    pr.mergeable === false ||
    pr.mergeable_state === "dirty" ||
    pr.mergeable_state === "conflicting"
  ) {
    return { label: "Conflicts", className: "bg-[#ffebe9] text-[#cf222e]" };
  }
  if (pr.state === "open") {
    return { label: "Open", className: "bg-[#1a7f37] text-white" };
  }
  return { label: "Closed", className: "bg-[#8250df] text-white" };
}

function reviewAuthor(review: Review): string {
  return review.reviewer?.login ?? "unknown";
}

function commentAuthor(comment: ReviewComment): string {
  return comment.author?.login ?? "unknown";
}

export default function PullRequestDetailPage({ params }: Props) {
  const router = useRouter();
  const [owner, setOwner] = useState("");
  const [repo, setRepo] = useState("");
  const [number, setNumber] = useState("");
  const [pr, setPr] = useState<PullRequest | null>(null);
  const [reviews, setReviews] = useState<Review[]>([]);
  const [reviewComments, setReviewComments] = useState<ReviewComment[]>([]);
  const [files, setFiles] = useState<PullRequestFile[]>([]);
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
      const prNumber = parseInt(number, 10);
      const [prData, reviewsData, commentsData, filesData] = await Promise.all([
        getPullRequest(owner, repo, prNumber),
        listReviews(owner, repo, prNumber).catch(() => [] as Review[]),
        listReviewComments(owner, repo, prNumber).catch(() => [] as ReviewComment[]),
        getPullRequestFiles(owner, repo, prNumber).catch(() => [] as PullRequestFile[]),
      ]);

      setPr(prData);
      setReviews(reviewsData);
      setReviewComments(commentsData);
      setFiles(filesData);
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) {
        router.push("/404");
        return;
      }
      setError(e instanceof Error ? e.message : "Failed to load pull request");
    } finally {
      setLoading(false);
    }
  }, [owner, repo, number, router]);

  useEffect(() => {
    loadPullRequest();
  }, [loadPullRequest]);

  const handleMerge = async (method: string) => {
    await mergePullRequest(owner, repo, parseInt(number, 10), {
      merge_method: method as "merge" | "squash" | "rebase",
    });
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
  const conversationCount = reviewComments.length + reviews.length + 1;

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
              <strong>{authorLogin(pr)}</strong> wants to merge changes into{" "}
              <code className="bg-[#f6f8fa] px-1.5 py-0.5 rounded text-xs">{pr.base_ref}</code> from{" "}
              <code className="bg-[#f6f8fa] px-1.5 py-0.5 rounded text-xs">{pr.head_ref}</code>
            </span>
            <span>· opened {formatAge(pr.created_at)}</span>
          </div>
        </div>

        <div className="flex gap-0 bg-white border border-b-0 border-[#d0d7de] rounded-t-lg px-4">
          {(
            [
              { id: "conversation" as const, label: "Conversation", count: conversationCount },
              { id: "commits" as const, label: "Commits", count: 0 },
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
                    {authorLogin(pr).slice(0, 2).toUpperCase()}
                  </span>
                  <span className="font-semibold text-sm">{authorLogin(pr)}</span>
                  <span className="text-xs text-[#656d76]">commented {formatAge(pr.created_at)}</span>
                </div>
                <div
                  className="prose prose-sm max-w-none pl-10"
                  dangerouslySetInnerHTML={{ __html: renderMarkdown(pr.body ?? "") }}
                />
              </div>

              {reviews.map((review) => (
                <div key={review.id} className="bg-white border border-[#d0d7de] rounded-md p-4 mb-4">
                  <div className="flex items-center gap-2 mb-3">
                    <span className="w-8 h-8 rounded-full bg-gradient-to-br from-[#0969da] to-[#8250df] text-white text-xs font-semibold inline-flex items-center justify-center">
                      {reviewAuthor(review).slice(0, 2).toUpperCase()}
                    </span>
                    <span className="font-semibold text-sm">{reviewAuthor(review)}</span>
                    <span className="text-xs text-[#656d76]">
                      {review.state.toLowerCase().replace("_", " ")}
                      {review.submitted_at ? ` ${formatAge(review.submitted_at)}` : ""}
                    </span>
                  </div>
                  {review.body ? (
                    <div
                      className="prose prose-sm max-w-none pl-10"
                      dangerouslySetInnerHTML={{ __html: renderMarkdown(review.body) }}
                    />
                  ) : null}
                </div>
              ))}

              {reviewComments.map((comment) => (
                <div key={comment.id} className="bg-white border border-[#d0d7de] rounded-md p-4 mb-4">
                  <div className="flex items-center gap-2 mb-3">
                    <span className="w-8 h-8 rounded-full bg-gradient-to-br from-[#0969da] to-[#8250df] text-white text-xs font-semibold inline-flex items-center justify-center">
                      {commentAuthor(comment).slice(0, 2).toUpperCase()}
                    </span>
                    <span className="font-semibold text-sm">{commentAuthor(comment)}</span>
                    <span className="text-xs text-[#656d76]">
                      reviewed {formatAge(comment.created_at)}
                      {comment.path
                        ? ` on ${comment.path}${comment.line ? `:${comment.line}` : ""}`
                        : ""}
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

              <MergePanel pr={toMergePanelPullRequest(pr)} onMerge={handleMerge} />
            </>
          )}

          {tab === "commits" && (
            <div className="divide-y divide-[#d0d7de]">
              <p className="text-[#656d76] py-4">Commit list will be available in a future update.</p>
            </div>
          )}

          {tab === "files" && (
            <div>
              {files.length === 0 ? (
                <p className="text-[#656d76] py-4">No file changes found.</p>
              ) : (
                files.map((file) => (
                  <DiffViewer
                    key={file.filename}
                    filename={file.filename}
                    patch={file.patch ?? undefined}
                    additions={file.additions}
                    deletions={file.deletions}
                  />
                ))
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
