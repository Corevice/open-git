"use client";

import { useState, type ReactNode } from "react";
import { Menu } from "lucide-react";

import { Header } from "@/components/layout/Header";
import { Sidebar } from "@/components/layout/Sidebar";
import { AuthProvider } from "@/lib/auth";

export function AppLayout({ children }: { children: ReactNode }) {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  return (
    <AuthProvider>
      <div className="min-h-screen bg-[color:var(--bg-base)]">
        <div className="sticky top-0 z-50 flex items-center bg-[#24292f]">
          <button
            type="button"
            className="shrink-0 px-3 py-3 text-white md:hidden"
            aria-label="Open navigation menu"
            onClick={() => setSidebarOpen(true)}
          >
            <Menu className="size-6" />
          </button>
          <div className="min-w-0 flex-1 [&_header]:static">
            <Header />
          </div>
        </div>
        <div className="flex">
          <Sidebar
            open={sidebarOpen}
            onClose={() => setSidebarOpen(false)}
          />
          <main className="flex-1">{children}</main>
        </div>
      </div>
    </AuthProvider>
  );
}
