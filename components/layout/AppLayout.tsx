'use client';

import type { ReactNode } from 'react';

import { Header } from '@/components/layout/Header';

export function AppLayout({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-[color:var(--bg-base)]">
      <Header />
      <main>{children}</main>
    </div>
  );
}
