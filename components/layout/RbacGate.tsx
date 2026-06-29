"use client";

import type { ReactNode } from "react";

export type RbacRole = "admin" | "write" | "read";

const ROLE_RANK: Record<RbacRole, number> = {
  read: 1,
  write: 2,
  admin: 3,
};

type RbacGateProps = {
  requiredRole: RbacRole;
  userRole: RbacRole | null | undefined;
  children: ReactNode;
};

export function RbacGate({
  requiredRole,
  userRole,
  children,
}: RbacGateProps) {
  if (
    userRole === null ||
    userRole === undefined ||
    ROLE_RANK[userRole] < ROLE_RANK[requiredRole]
  ) {
    return null;
  }

  return children;
}
