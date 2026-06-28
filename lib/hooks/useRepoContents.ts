import useSWR from "swr";

import { createRepoApiClient, isApiError } from "@/lib/api-client";
import { env } from "@/lib/env";

const apiClient = createRepoApiClient(env.NEXT_PUBLIC_API_BASE_URL);

export interface RepoContentFile {
  name: string;
  path: string;
  sha: string;
  size?: number;
  type: "file" | "dir";
  content?: string | null;
  encoding?: string;
  download_url?: string;
  truncated?: boolean;
}

export type RepoContentsData = RepoContentFile | RepoContentFile[];

export interface RepoMetadata {
  name: string;
  description: string | null;
  private: boolean;
  visibility?: string;
  default_branch: string;
  stargazers_count: number;
  watchers_count: number;
  forks_count: number;
  owner: { login: string };
}

export interface BranchItem {
  name: string;
}

function decodePathSegment(segment: string): string | null {
  try {
    return decodeURIComponent(segment);
  } catch {
    return null;
  }
}

export function sanitizeRepoPath(path: string): string | null {
  if (path === "") return "";

  const parts: string[] = [];
  for (const rawSegment of path.split("/")) {
    if (rawSegment.length === 0) continue;

    const segment = decodePathSegment(rawSegment);
    if (segment === null || segment === "." || segment === "..") {
      return null;
    }
    parts.push(segment);
  }

  return parts.join("/");
}

export function sanitizeRepoRef(ref: string): string {
  const trimmed = ref.trim();
  if (!trimmed || trimmed.includes("\\")) return "";

  const decoded = decodePathSegment(trimmed);
  if (decoded === null || decoded.includes("..")) return "";

  for (const segment of decoded.split("/")) {
    if (segment === "." || segment === "..") return "";
  }

  return trimmed;
}

export function useRepoContents(
  owner: string,
  repo: string,
  path: string | null,
  ref: string,
) {
  const safePath = path === null ? null : sanitizeRepoPath(path);
  const safeRef = sanitizeRepoRef(ref);
  const key =
    owner && repo && safeRef && safePath !== null
      ? ([owner, repo, safePath, safeRef] as const)
      : null;

  const { data, isLoading, error } = useSWR(
    key,
    ([o, r, p, f]) => apiClient.getContents<RepoContentsData>(o, r, p, f),
    { revalidateOnFocus: false },
  );

  const isNotFound = isApiError(error) && error.status === 404;

  return { data, isLoading, error: error ?? null, isNotFound };
}

export function useRepoMetadata(owner: string, repo: string) {
  const key = owner && repo ? (["repo-metadata", owner, repo] as const) : null;

  const { data, isLoading, error } = useSWR(
    key,
    ([, o, r]) => apiClient.getRepo<RepoMetadata>(o, r),
    { revalidateOnFocus: false },
  );

  const isNotFound = isApiError(error) && error.status === 404;

  return { data, isLoading, error: error ?? null, isNotFound };
}

export function useRepoBranches(owner: string, repo: string, fallbackBranch: string) {
  const key = owner && repo ? (["repo-branches", owner, repo] as const) : null;

  const { data, isLoading, error } = useSWR(
    key,
    async ([, o, r]) => {
      const branches = await apiClient.getBranches<BranchItem[]>(o, r);
      return branches.length > 0 ? branches : [{ name: fallbackBranch }];
    },
    { revalidateOnFocus: false },
  );

  return {
    branches: data ?? [{ name: fallbackBranch }],
    isLoading,
    error: error ?? null,
  };
}
