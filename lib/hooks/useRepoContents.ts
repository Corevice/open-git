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

export function sanitizeRepoPath(path: string): string {
  return path
    .split("/")
    .filter((segment) => segment.length > 0 && segment !== "." && segment !== "..")
    .join("/");
}

export function sanitizeRepoRef(ref: string): string {
  const trimmed = ref.trim();
  if (!trimmed || trimmed.includes("..") || /[/\\]/.test(trimmed)) {
    return "";
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

  return { data, isLoading, error, isNotFound };
}
