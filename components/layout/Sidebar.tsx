"use client";

import Link from "next/link";

const navLinks = [
  { href: "/04-dashboard", label: "Dashboard" },
  { href: "/05-repo-list", label: "Repositories" },
  { href: "/12-issue-list", label: "Issues" },
  { href: "/13-pr-list", label: "Pull Requests" },
];

type SidebarProps = {
  open: boolean;
  onClose: () => void;
};

function NavLinks({ onNavigate }: { onNavigate?: () => void }) {
  return (
    <nav className="flex flex-col gap-1 p-4">
      {navLinks.map((link) => (
        <Link
          key={link.href}
          href={link.href}
          onClick={onNavigate}
          className="rounded-md px-3 py-2 text-sm text-[#c9d1d9] transition-colors hover:bg-[#30363d] hover:text-white"
        >
          {link.label}
        </Link>
      ))}
    </nav>
  );
}

export function Sidebar({ open, onClose }: SidebarProps) {
  return (
    <>
      {open && (
        <div className="fixed inset-0 z-40 md:hidden">
          <button
            type="button"
            aria-label="Close navigation menu"
            className="absolute inset-0 bg-black/50"
            data-testid="sidebar-backdrop"
            onClick={onClose}
          />
          <aside
            aria-label="Navigation"
            data-testid="sidebar-drawer"
            className="relative z-10 h-full w-64 border-r border-[#30363d] bg-[#161b22]"
          >
            <NavLinks onNavigate={onClose} />
          </aside>
        </div>
      )}

      <aside
        aria-label="Navigation"
        className="hidden w-64 shrink-0 border-r border-[#d0d7de] bg-white md:block"
      >
        <NavLinks />
      </aside>
    </>
  );
}
