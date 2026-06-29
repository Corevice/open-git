import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import {
  genericActionError,
  resolveOrgLogin,
  verifyOrgAdminRole,
} from "@/lib/admin/security";

export type OrgAdminAccessResult =
  | { status: "ok"; token: string; org: string }
  | { status: "unauthenticated" }
  | { status: "access_denied" }
  | { status: "error"; message: string };

export async function getAuthTokenFromCookies(): Promise<string | null> {
  const cookieStore = await cookies();
  return cookieStore.get("authToken")?.value ?? null;
}

export async function requireAuthToken(): Promise<string> {
  const token = await getAuthTokenFromCookies();
  if (!token) {
    redirect("/login");
  }
  return token;
}

export async function requireOrgAdminAccess(
  orgParam?: string | null,
): Promise<OrgAdminAccessResult> {
  const token = await getAuthTokenFromCookies();
  if (!token) {
    return { status: "unauthenticated" };
  }

  const orgResult = await resolveOrgLogin(token, orgParam);
  if (!orgResult.ok) {
    return { status: "error", message: orgResult.message };
  }

  const roleCheck = await verifyOrgAdminRole(token, orgResult.org);
  if (roleCheck === "unauthorized") {
    return { status: "unauthenticated" };
  }
  if (roleCheck === "forbidden") {
    return { status: "access_denied" };
  }
  if (roleCheck === "error") {
    return { status: "error", message: genericActionError("verify admin access") };
  }

  return { status: "ok", token, org: orgResult.org };
}
