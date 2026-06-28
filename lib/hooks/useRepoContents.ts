import useSWR from "swr";

import { createRepoApiClient } from "@/lib/api-client";
import { env } from "@/lib/env";

const apiClient = createRepoApiClient(env.NEXT_PUBLIC_API_BASE_URL);

export function useRepoContents(
  owner: string,
  repo: string,
  path: string,
  ref: string,
) {
  const { data, isLoading, error } = useSWR(
    [owner, repo, path, ref],
    ([o, r, p, f]) => apiClient.getContents(o, r, p, f),
    { revalidateOnFocus: false },
  );

  return { data, isLoading, error };
}
