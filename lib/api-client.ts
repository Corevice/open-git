export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export class ApiClient {
  private pat: string | null = null;

  constructor(private baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "") {}

  setPat(pat: string | null): void {
    this.pat = pat;
  }

  private async fetch<T>(
    path: string,
    init?: RequestInit,
  ): Promise<T | null> {
    const headers = new Headers(init?.headers);

    if (init?.body && !headers.has("Content-Type")) {
      headers.set("Content-Type", "application/json");
    }

    if (this.pat) {
      headers.set("Authorization", `Bearer ${this.pat}`);
    }

    const response = await fetch(`${this.baseUrl}${path}`, {
      ...init,
      headers,
    });

    if (response.status === 404) {
      return null;
    }

    if (!response.ok) {
      let message = response.statusText;
      try {
        const body = (await response.json()) as { message?: string };
        message = body.message ?? message;
      } catch {
        // ignore JSON parse errors
      }
      throw new ApiError(response.status, message);
    }

    if (response.status === 204) {
      return null;
    }

    const contentType = response.headers.get("content-type");
    if (contentType?.includes("application/json")) {
      return response.json() as Promise<T>;
    }

    return null;
  }

  getRepo(owner: string, repo: string): Promise<unknown | null> {
    return this.fetch(`/repos/${owner}/${repo}`);
  }

  listRepos(page = 1, perPage = 30): Promise<unknown | null> {
    return this.fetch(`/user/repos?page=${page}&per_page=${perPage}`);
  }

  createRepo(
    name: string,
    visibility: "public" | "private",
    description?: string,
  ): Promise<unknown | null> {
    return this.fetch("/user/repos", {
      method: "POST",
      body: JSON.stringify({
        name,
        visibility,
        ...(description !== undefined ? { description } : {}),
      }),
    });
  }

  deleteRepo(owner: string, repo: string): Promise<unknown | null> {
    return this.fetch(`/repos/${owner}/${repo}`, { method: "DELETE" });
  }

  getContents(
    owner: string,
    repo: string,
    path?: string,
    ref?: string,
  ): Promise<unknown | null> {
    const params = new URLSearchParams();
    if (path !== undefined) {
      params.set("path", path);
    }
    if (ref !== undefined) {
      params.set("ref", ref);
    }
    const query = params.toString();
    return this.fetch(
      `/repos/${owner}/${repo}/contents${query ? `?${query}` : ""}`,
    );
  }

  getCommits(
    owner: string,
    repo: string,
    sha?: string,
    page?: number,
  ): Promise<unknown | null> {
    const params = new URLSearchParams();
    if (sha !== undefined) {
      params.set("sha", sha);
    }
    if (page !== undefined) {
      params.set("page", String(page));
    }
    const query = params.toString();
    return this.fetch(
      `/repos/${owner}/${repo}/commits${query ? `?${query}` : ""}`,
    );
  }

  getBranches(owner: string, repo: string): Promise<unknown | null> {
    return this.fetch(`/repos/${owner}/${repo}/branches`);
  }

  getTags(owner: string, repo: string): Promise<unknown | null> {
    return this.fetch(`/repos/${owner}/${repo}/tags`);
  }

  compareRefs(
    owner: string,
    repo: string,
    base: string,
    head: string,
  ): Promise<unknown | null> {
    return this.fetch(`/repos/${owner}/${repo}/compare/${base}...${head}`);
  }
}

export const apiClient = new ApiClient();
