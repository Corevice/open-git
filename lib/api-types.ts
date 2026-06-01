export interface User {
  id: number;
  login: string;
  email: string;
}

export interface Repository {
  id: number;
  name: string;
  owner: string;
  visibility: string;
  defaultBranch: string;
}

export interface Issue {
  id: number;
  number: number;
  title: string;
  body: string;
  state: string;
  author: string;
}

export interface PullRequest {
  id: number;
  number: number;
  headRef: string;
  baseRef: string;
  state: string;
  mergedAt: string | null;
}

export interface Token {
  id: number;
  name: string;
  scopes: string[];
  createdAt: string;
}

export interface OAuthApp {
  id: number;
  name: string;
  clientId: string;
}
