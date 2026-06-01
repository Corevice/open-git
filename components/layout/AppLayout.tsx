"use client";

import type { ReactNode } from "react";

import { Header } from "@/components/layout/Header";
import { AuthProvider } from "@/lib/auth";

export function AppLayout({ children }: { children: ReactNode }) {
  return (
    <AuthProvider>
      <div className="min-h-screen bg-[color:var(--bg-base)]">
        <Header />
        <main>{children}</main>
      </div>
    </AuthProvider>
  );
}
