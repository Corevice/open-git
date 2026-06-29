export const API_BASE = "/api/v3";

export interface GitHubUser {
  id: number;
  login: string;
  email: string;
  type: string;
}

export interface GitHubRepo {
  id: number;
  name: string;
  full_name: string;
  private: boolean;
  description: string;
  default_branch: string;
  owner: { login: string; id: number };
}

export interface GitHubOrg {
  id: number;
  login: string;
  name: string;
  type: string;
}

export function createGitHubClient(pat: string) {
  async function fetch<T>(path: string, init?: RequestInit): Promise<T> {
    const url = `${API_BASE}${path}`;
    const headers = new Headers(init?.headers);
    headers.set("Authorization", `token ${pat}`);
    headers.set("Accept", "application/vnd.github.v3+json");

    const response = await globalThis.fetch(url, {
      ...init,
      headers,
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    return response.json() as Promise<T>;
  }

  return {
    fetch,
    getUser: (): Promise<GitHubUser> => fetch<GitHubUser>("/user"),
    getRepo: (owner: string, repo: string): Promise<GitHubRepo> =>
      fetch<GitHubRepo>(`/repos/${owner}/${repo}`),
    listUserRepos: (
      page?: number,
      perPage?: number,
    ): Promise<GitHubRepo[]> => {
      const params = new URLSearchParams();
      if (page !== undefined) {
        params.set("page", String(page));
      }
      if (perPage !== undefined) {
        params.set("per_page", String(perPage));
      }
      const query = params.toString();
      return fetch<GitHubRepo[]>(
        `/user/repos${query ? `?${query}` : ""}`,
      );
    },
    listUserOrgs: (): Promise<GitHubOrg[]> =>
      fetch<GitHubOrg[]>("/user/orgs"),
  };
}
