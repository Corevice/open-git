import Link from "next/link";
import type { ReactNode } from "react";

type SecurityPageLayoutProps = {
  title: string;
  breadcrumbSuffix?: string;
  backHref?: string;
  backLabel?: string;
  children: ReactNode;
};

export function SecurityPageLayout({
  title,
  breadcrumbSuffix,
  backHref,
  backLabel,
  children,
}: SecurityPageLayoutProps) {
  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <header className="sticky top-0 z-50 flex h-16 items-center justify-between border-b border-[#d1d9e0] bg-white/85 px-6 backdrop-blur">
        <Link
          href="/dashboard"
          className="flex items-center gap-2 text-lg font-extrabold"
        >
          <span className="text-xl">🐙</span>
          <span>OpenHub</span>
        </Link>
        {backHref && backLabel ? (
          <Link
            href={backHref}
            className="rounded-md px-3 py-1.5 text-sm hover:bg-[#f6f8fa]"
          >
            {backLabel}
          </Link>
        ) : null}
      </header>

      <div className="mx-auto max-w-[1200px] px-6 py-6">
        <div className="mb-4 text-sm text-[#656d76]">
          <Link href="/dashboard" className="text-[#0969da]">
            Dashboard
          </Link>{" "}
          /{" "}
          {breadcrumbSuffix ? (
            <>
              <Link href="/admin/security" className="text-[#0969da]">
                Admin / Security
              </Link>{" "}
              / {breadcrumbSuffix}
            </>
          ) : (
            "Admin / Security"
          )}
        </div>
        <h1 className="mb-6 text-2xl font-semibold">{title}</h1>
        {children}
      </div>
    </div>
  );
}

type SecurityAccessDeniedProps = {
  title: string;
  breadcrumbSuffix?: string;
  message?: string;
};

export function SecurityAccessDenied({
  title,
  breadcrumbSuffix,
  message = "You do not have permission to view this page. Admin access is required.",
}: SecurityAccessDeniedProps) {
  return (
    <SecurityPageLayout title={title} breadcrumbSuffix={breadcrumbSuffix}>
      <div className="rounded-md border border-[#d0d7de] bg-white p-6">
        <p className="text-sm font-semibold text-[#cf222e]">Access Denied</p>
        <p className="mt-2 text-sm text-[#656d76]">{message}</p>
      </div>
    </SecurityPageLayout>
  );
}
