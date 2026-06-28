import type { AdvisorySeverity } from "@/lib/api-types";

export function getSecurityApiBase(): string {
  return (
    process.env.NEXT_PUBLIC_API_BASE_URL ??
    process.env.NEXT_PUBLIC_API_URL ??
    ""
  );
}

const ORG_LOGIN_PATTERN = /^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,38}[a-zA-Z0-9])?$/;

export function sanitizeOrgLogin(org: string | null | undefined): string | null {
  if (!org) {
    return null;
  }

  const trimmed = org.trim();
  if (!ORG_LOGIN_PATTERN.test(trimmed)) {
    return null;
  }

  return trimmed;
}

export function buildOrgQueryString(org: string | null | undefined): string {
  const safeOrg = sanitizeOrgLogin(org);
  return safeOrg ? `?org=${encodeURIComponent(safeOrg)}` : "";
}

export function isAdminOrgRole(role: string): boolean {
  const normalized = role.toLowerCase();
  return normalized === "admin" || normalized === "owner";
}

export function sanitizeAuditSearchPhrase(phrase: string): string {
  return phrase
    .trim()
    .replace(/[\u0000-\u001f\u007f]/g, "")
    .slice(0, 256);
}

export function datetimeLocalToIso(value: string): string {
  const match = value.match(/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/);
  if (!match) {
    return "";
  }

  const [, year, month, day, hour, minute] = match;
  const date = new Date(
    Number(year),
    Number(month) - 1,
    Number(day),
    Number(hour),
    Number(minute),
  );

  if (Number.isNaN(date.getTime())) {
    return "";
  }

  return date.toISOString();
}

export function maskIpAddress(ip: string | null): string {
  if (!ip) {
    return "—";
  }

  const ipv4Match = ip.match(/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/);
  if (ipv4Match) {
    return `${ipv4Match[1]}.${ipv4Match[2]}.*.*`;
  }

  if (ip.includes(":")) {
    const [firstSegment] = ip.split(":");
    return firstSegment ? `${firstSegment}:****` : "—";
  }

  return "—";
}

export function genericActionError(action: string): string {
  return `Unable to ${action}. Please try again.`;
}

const severityBadgeClass: Record<AdvisorySeverity, string> = {
  critical: "border-transparent bg-[#cf222e] text-white",
  high: "border-transparent bg-[#ffebe9] text-[#cf222e]",
  medium: "border-transparent bg-[#fff8c5] text-[#9a6700]",
  low: "border-transparent bg-[#eaeef2] text-[#656d76]",
};

export function getSeverityBadgeClass(
  severity: string,
): string {
  if (severity in severityBadgeClass) {
    return severityBadgeClass[severity as AdvisorySeverity];
  }

  return severityBadgeClass.low;
}

async function authFetch(
  token: string,
  path: string,
  init?: RequestInit,
): Promise<Response> {
  return fetch(`${getSecurityApiBase()}${path}`, {
    ...init,
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${token}`,
      ...(init?.headers ?? {}),
    },
    cache: "no-store",
  });
}

export async function resolveOrgLogin(
  token: string,
  orgParam?: string | null,
): Promise<{ ok: true; org: string } | { ok: false; message: string }> {
  const sanitized = sanitizeOrgLogin(orgParam);
  if (sanitized) {
    return { ok: true, org: sanitized };
  }

  try {
    const response = await authFetch(token, "/api/v3/user/orgs");
    if (response.status === 401) {
      return { ok: false, message: "Authentication required" };
    }
    if (!response.ok) {
      return {
        ok: false,
        message: `Failed to load organizations (${response.status})`,
      };
    }

    const orgs = (await response.json()) as { login: string }[];
    if (orgs.length === 0) {
      return { ok: false, message: "No organization found" };
    }

    return { ok: true, org: orgs[0].login };
  } catch {
    return { ok: false, message: "Failed to load organizations" };
  }
}

export async function verifyOrgAdminRole(
  token: string,
  org: string,
): Promise<"admin" | "forbidden" | "unauthorized" | "error"> {
  try {
    const [userResponse, membersResponse] = await Promise.all([
      authFetch(token, "/api/v3/user"),
      authFetch(token, `/api/v3/orgs/${encodeURIComponent(org)}/members`),
    ]);

    if (userResponse.status === 401 || membersResponse.status === 401) {
      return "unauthorized";
    }

    if (membersResponse.status === 403 || membersResponse.status === 404) {
      return "forbidden";
    }

    if (!userResponse.ok || !membersResponse.ok) {
      return "error";
    }

    const user = (await userResponse.json()) as { login: string };
    const members = (await membersResponse.json()) as {
      login: string;
      role: string;
    }[];
    const membership = members.find((member) => member.login === user.login);

    if (!membership || !isAdminOrgRole(membership.role)) {
      return "forbidden";
    }

    return "admin";
  } catch {
    return "error";
  }
}

export async function checkClientOrgAdminAccess(options: {
  token: string | null;
  orgParam: string | null;
  onUnauthenticated: () => void;
}): Promise<
  | { status: "ok"; org: string }
  | { status: "unauthenticated" }
  | { status: "access_denied" }
  | { status: "error"; message: string }
> {
  if (!options.token) {
    options.onUnauthenticated();
    return { status: "unauthenticated" };
  }

  const orgResult = await resolveOrgLogin(options.token, options.orgParam);
  if (!orgResult.ok) {
    return { status: "error", message: orgResult.message };
  }

  const roleCheck = await verifyOrgAdminRole(options.token, orgResult.org);
  if (roleCheck === "unauthorized") {
    options.onUnauthenticated();
    return { status: "unauthenticated" };
  }
  if (roleCheck === "forbidden") {
    return { status: "access_denied" };
  }
  if (roleCheck === "error") {
    return { status: "error", message: genericActionError("verify admin access") };
  }

  return { status: "ok", org: orgResult.org };
}
