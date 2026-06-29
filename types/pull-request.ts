export interface PullRequestUser {
  login: string;
  id?: string | number;
}

export interface PullRequest {
  id: string;
  number: number;
  title: string;
  body: string;
  state: "open" | "closed";
  draft: boolean;
  head_ref: string;
  base_ref: string;
  head_sha: string;
  base_sha: string;
  merge_commit_sha: string | null;
  mergeable: boolean | null;
  mergeable_state: string;
  merged_at: string | null;
  merged_by: string | null;
  author_id: string;
  created_at: string;
  updated_at: string;
  user?: PullRequestUser;
  labels?: { name: string; color: string }[];
}

export interface PullRequestFile {
  filename: string;
  previous_filename: string | null;
  status: string;
  additions: number;
  deletions: number;
  patch: string | null;
  binary: boolean;
  sha: string | null;
}

export interface Review {
  id: string;
  state: string;
  body: string;
  submitted_at: string | null;
  reviewer: PullRequestUser | null;
}

export interface ReviewComment {
  id: string;
  path: string;
  line: number | null;
  side: string | null;
  body: string;
  diff_hunk: string | null;
  author: PullRequestUser;
  in_reply_to_id: string | null;
  resolved: boolean;
  created_at: string;
}

export interface PullRequestsResponse {
  items: PullRequest[];
  total: number;
}

export interface CreatePullRequestInput {
  title: string;
  head: string;
  base: string;
  body?: string;
  draft?: boolean;
}

export interface MergeInput {
  merge_method?: "merge" | "squash" | "rebase";
  commit_title?: string;
  commit_message?: string;
  sha?: string;
}

export interface MergeResponse {
  sha?: string;
  merged: boolean;
  message: string;
}

export interface CreateReviewInput {
  event: "APPROVE" | "REQUEST_CHANGES" | "COMMENT";
  body?: string;
  comments?: Array<{
    path: string;
    line: number;
    body: string;
  }>;
}

export interface UpdatePullRequestInput {
  title?: string;
  body?: string;
  state?: "open" | "closed";
  base?: string;
}
