"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";
import { Bell, ChevronDown, Search } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ThemeToggle } from "@/components/ui/theme-toggle";
import { getCurrentUser, getOrgs } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/utils";

const navLinks = [
  { href: "/dashboard", label: "Dashboard" },
  { href: "/", label: "Repositories" },
];

type OrgSummary = { login: string; avatar_url: string };

function getInitials(login: string): string {
  return login.slice(0, 2).toUpperCase();
}

function getSearchApiBaseUrl(): string {
  return (
    process.env.NEXT_PUBLIC_API_BASE_URL ??
    process.env.NEXT_PUBLIC_API_URL ??
    "http://localhost:8080"
  );
}

export function Header() {
  const router = useRouter();
  const { token, isAuthenticated, logout } = useAuth();
  const [search, setSearch] = useState("");
  const [menuOpen, setMenuOpen] = useState(false);
  const [user, setUser] = useState<{ login: string; avatar_url: string } | null>(
    null,
  );
  const [orgs, setOrgs] = useState<OrgSummary[]>([]);
  const menuRef = useRef<HTMLDivElement>(null);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (!isAuthenticated || !token) {
      setUser(null);
      setOrgs([]);
      return;
    }

    let cancelled = false;

    async function loadAuthData() {
      try {
        const [currentUser, orgList] = await Promise.all([
          getCurrentUser(token),
          getOrgs(token),
        ]);
        if (!cancelled) {
          setUser(currentUser);
          setOrgs(orgList);
        }
      } catch {
        if (!cancelled) {
          setUser(null);
          setOrgs([]);
        }
      }
    }

    void loadAuthData();

    return () => {
      cancelled = true;
    };
  }, [isAuthenticated, token]);

  const handleSearch = useCallback(
    (query: string) => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }

      debounceTimerRef.current = setTimeout(() => {
        const trimmed = query.trim();
        if (trimmed === "") {
          return;
        }

        const headers: Record<string, string> = {
          "Content-Type": "application/json",
        };
        if (token) {
          headers.Authorization = `Bearer ${token}`;
        }

        void fetch(
          `${getSearchApiBaseUrl()}/api/v3/search/repositories?q=${encodeURIComponent(trimmed)}`,
          { headers },
        );
      }, 300);
    },
    [token],
  );

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = search.trim();
    if (trimmed === "") {
      return;
    }
    router.push(`/search?q=${encodeURIComponent(trimmed)}`);
  };

  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

  const avatarInitials = user ? getInitials(user.login) : "";

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
        onSubmit={handleSearchSubmit}
        className="relative mx-4 hidden max-w-md flex-1 md:block"
      >
        <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[#8b949e]" />
        <Input
          type="search"
          placeholder="Search or jump to..."
          value={search}
          onChange={(e) => {
            const value = e.target.value;
            setSearch(value);
            handleSearch(value);
          }}
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
        <ThemeToggle />

        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="text-[#c9d1d9] hover:bg-[#30363d] hover:text-white"
          aria-label="Notifications"
        >
          <Bell className="size-5" />
        </Button>

        {isAuthenticated ? (
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
              aria-label="User menu"
            >
              <span className="flex size-8 items-center justify-center rounded-full bg-gradient-to-br from-indigo-600 to-violet-600 text-xs font-semibold">
                {avatarInitials}
              </span>
              <ChevronDown className="mr-1 size-4 text-[#8b949e]" />
            </button>

            {menuOpen && (
              <div
                role="menu"
                className="absolute right-0 mt-2 w-48 overflow-hidden rounded-md border border-[#30363d] bg-[#161b22] py-1 shadow-lg"
              >
                {orgs.length > 0 && (
                  <div className="border-b border-[#30363d] px-4 py-2">
                    <p className="mb-1 text-xs font-semibold uppercase tracking-wide text-[#8b949e]">
                      Organizations
                    </p>
                    <ul role="list">
                      {orgs.map((org) => (
                        <li key={org.login}>
                          <Link
                            href={`/${org.login}`}
                            role="menuitem"
                            className="block rounded px-2 py-1.5 text-sm text-[#c9d1d9] hover:bg-[#30363d] hover:text-white"
                            onClick={() => setMenuOpen(false)}
                          >
                            {org.login}
                          </Link>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
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
        ) : (
          <Link
            href="/login"
            className="rounded-md px-3 py-2 text-sm font-medium text-[#c9d1d9] transition-colors hover:bg-[#30363d] hover:text-white"
          >
            Sign in
          </Link>
        )}
      </div>
    </header>
  );
}
