"use client";

import Link from "next/link";

const navLinks = [
  { href: "/dashboard", label: "Dashboard" },
  { href: "/dashboard", label: "Repositories" },
  { href: "/dashboard", label: "Issues" },
  { href: "/dashboard", label: "Pull Requests" },
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
          key={link.label}
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
        <>
          <div
            className="fixed inset-0 z-40 bg-black/50 md:hidden"
            data-testid="sidebar-backdrop"
            aria-hidden="true"
            onClick={onClose}
          />
          <aside
            aria-label="Navigation"
            data-testid="sidebar-drawer"
            className="fixed left-0 top-0 z-50 h-full w-64 border-r border-[#30363d] bg-[#161b22] md:hidden"
          >
            <NavLinks onNavigate={onClose} />
          </aside>
        </>
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
