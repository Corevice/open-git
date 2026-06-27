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
  clone_url?: string;
  ssh_url?: string;
}

export interface SSHKey {
  id: string;
  title: string;
  key_type: string;
  fingerprint: string;
  created_at: string;
  last_used_at: string | null;
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
