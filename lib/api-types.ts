export interface User {
  id: number;
  login: string;
  email: string;
  name?: string;
  bio?: string;
  avatar_url?: string;
}

export interface OrgProfile {
  id: number;
  login: string;
  name: string;
  description: string;
  type: "Organization";
}

export interface OrgMember {
  id: number;
  login: string;
  role: string;
}

export interface AccessTokenMeta {
  id: number;
  note: string;
  scopes: string[];
  expires_at: string | null;
  created_at: string;
  last_used_at: string | null;
}

export interface CreateTokenResult extends AccessTokenMeta {
  token: string;
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

export type OAuthApp = {
  id: string;
  client_id: string;
  name: string;
  homepage_url: string;
  callback_urls: string[];
  owner_type: string;
  created_at: string;
};

export type OAuthAppWithSecret = OAuthApp & { client_secret: string };

export type OAuthAppCreateInput = {
  name: string;
  homepage_url: string;
  callback_urls: string[];
  owner_type: "user" | "organization";
  owner_user_id?: number;
  organization_id?: number;
};

export type OAuthAuthorizationInfo = {
  oauth_app_id: string;
  app_name: string;
  granted_scopes: string[];
  updated_at: string;
};

export interface ObservabilityDashboard {
  uid: string;
  title: string;
  category: "system" | "git" | "api" | "ci" | "db";
  grafana_path: string;
}

export interface ObservabilityDashboardsResponse {
  dashboards: ObservabilityDashboard[];
}

export interface GrafanaURLResponse {
  url: string;
}

export type AdvisorySeverity = "critical" | "high" | "medium" | "low";

export type AdvisoryState = "open" | "acknowledged" | "resolved" | "dismissed";

export type DismissedReason =
  | "no_bandwidth"
  | "tolerable_risk"
  | "inaccurate"
  | "not_used";

export interface SecurityAdvisory {
  id: string;
  organization_id: string;
  repository_id: string | null;
  ghsa_id: string;
  cve_id: string | null;
  severity: AdvisorySeverity;
  summary: string;
  description: string;
  affected_package: string;
  affected_versions: string;
  patched_versions: string;
  state: AdvisoryState;
  dismissed_reason: DismissedReason | null;
  created_at: string;
  updated_at: string;
}

export interface DependabotAlert {
  id: string;
  organization_id: string;
  repository_id: string;
  alert_number: number;
  advisory_id: string;
  manifest_path: string;
  state: "open" | "dismissed" | "fixed";
  auto_dismissed_at: string | null;
}

export interface AuditLogEntry {
  id: string;
  organization_id: string;
  actor_id: string | null;
  action: string;
  resource_type: string;
  resource_id: string;
  metadata: Record<string, unknown>;
  ip_address: string | null;
  user_agent: string | null;
  created_at: string;
}

export interface ScanJob {
  id: string;
  organization_id: string;
  repository_id: string;
  type: "dependency" | "secret";
  status: "queued" | "running" | "completed" | "scan_failed" | "parse_error";
  retry_count: number;
  started_at: string | null;
  finished_at: string | null;
  error: string | null;
}

export type ActionCompatibilityStatus =
  | "supported"
  | "partial"
  | "unsupported"
  | "unknown";

export interface ActionCompatibilityResult {
  action: string;
  version: string;
  status: ActionCompatibilityStatus;
  note: string | null;
  last_verified_at: string | null;
}
