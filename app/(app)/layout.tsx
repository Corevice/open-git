"use client";

import type { ReactNode } from "react";

import { AppLayout } from "@/components/layout/AppLayout";
import { AuthProvider } from "@/lib/auth-context";

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <AuthProvider>
      <AppLayout>{children}</AppLayout>
    </AuthProvider>
  );
}
