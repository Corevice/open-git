import type { ReactNode } from "react";

export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-[color:var(--bg-base)] p-6">
      <div className="w-full max-w-md">{children}</div>
    </div>
  );
}
