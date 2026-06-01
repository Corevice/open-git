"use client";

import Link from "next/link";
import { useRef, useState } from "react";
import { Bell, ChevronDown, Search } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/utils";

const navLinks = [
  { href: "/04-dashboard", label: "Dashboard" },
  { href: "/05-repo-list", label: "Repositories" },
  { href: "/12-issue-list", label: "Issues" },
  { href: "/13-pr-list", label: "Pull requests" },
];

export function Header() {
  const { logout } = useAuth();
  const [search, setSearch] = useState("");
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
  };

  return (
    <header className="sticky top-0 z-50 flex items-center justify-between gap-4 border-b border-[#30363d] bg-[#24292f] px-4 py-3 text-white">
      <Link
        href="/"
        className="flex shrink-0 items-center gap-2 font-semibold text-white hover:text-white"
      >
        <span aria-hidden>🐙</span>
        <span>OpenHub</span>
      </Link>

      <form
        onSubmit={handleSearch}
        className="relative mx-4 hidden max-w-md flex-1 md:block"
      >
        <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[#8b949e]" />
        <Input
          type="search"
          placeholder="Search or jump to..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="h-9 border-[#30363d] bg-[#0d1117] pl-9 text-sm text-white placeholder:text-[#8b949e] focus-visible:ring-[#6366f1]"
        />
      </form>

      <nav className="hidden items-center gap-1 lg:flex">
        {navLinks.map((link) => (
          <Link
            key={link.href}
            href={link.href}
            className="rounded-md px-3 py-2 text-sm text-[#c9d1d9] transition-colors hover:bg-[#30363d] hover:text-white"
          >
            {link.label}
          </Link>
        ))}
      </nav>

      <div className="flex shrink-0 items-center gap-2">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="text-[#c9d1d9] hover:bg-[#30363d] hover:text-white"
          aria-label="Notifications"
        >
          <Bell className="size-5" />
        </Button>

        <div className="relative" ref={menuRef}>
          <button
            type="button"
            onClick={() => setMenuOpen((open) => !open)}
            className={cn(
              "flex items-center gap-1 rounded-full border border-[#30363d] p-0.5 transition-colors hover:border-[#8b949e]",
              menuOpen && "border-[#8b949e]",
            )}
            aria-expanded={menuOpen}
            aria-haspopup="menu"
          >
            <span className="flex size-8 items-center justify-center rounded-full bg-gradient-to-br from-indigo-600 to-violet-600 text-xs font-semibold">
              YT
            </span>
            <ChevronDown className="mr-1 size-4 text-[#8b949e]" />
          </button>

          {menuOpen && (
            <div
              role="menu"
              className="absolute right-0 mt-2 w-48 overflow-hidden rounded-md border border-[#30363d] bg-[#161b22] py-1 shadow-lg"
            >
              <Link
                href="/15-settings"
                role="menuitem"
                className="block px-4 py-2 text-sm text-[#c9d1d9] hover:bg-[#30363d] hover:text-white"
                onClick={() => setMenuOpen(false)}
              >
                Settings
              </Link>
              <button
                type="button"
                role="menuitem"
                className="w-full px-4 py-2 text-left text-sm text-[#c9d1d9] hover:bg-[#30363d] hover:text-white"
                onClick={() => {
                  logout();
                  setMenuOpen(false);
                }}
              >
                Sign out
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
