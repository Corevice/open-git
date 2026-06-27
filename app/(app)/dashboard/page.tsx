"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { apiClient } from "@/lib/api-client";

type Repo = {
  name: string;
  owner: string;
  visibility: string;
  default_branch: string;
  created_at: string;
};

function visibilityBadgeClass(visibility: string): string {
  const v = visibility.toLowerCase();
  if (v === "public") return "bg-[color:var(--success-light)] text-[color:var(--success)]";
  if (v === "private") return "bg-[color:var(--warning-light)] text-[color:var(--warning)]";
  return "bg-[color:var(--info-light)] text-[color:var(--info)]";
}

function visibilityLabel(visibility: string): string {
  return visibility.charAt(0).toUpperCase() + visibility.slice(1);
}

function formatCreated(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return date.toLocaleDateString("ja-JP");
}

function RepoCardSkeleton() {
  return (
    <div className="rounded-lg border border-[color:var(--border)] bg-white p-4">
      <div className="mb-2 h-5 w-1/2 animate-pulse rounded bg-[color:var(--bg-muted)]" />
      <div className="mb-3 h-4 w-16 animate-pulse rounded-full bg-[color:var(--bg-muted)]" />
      <div className="h-3 w-24 animate-pulse rounded bg-[color:var(--bg-muted)]" />
      <div className="mt-2 h-3 w-32 animate-pulse rounded bg-[color:var(--bg-muted)]" />
    </div>
  );
}

export default function DashboardPage() {
  const [repos, setRepos] = useState<Repo[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function loadRepos() {
      try {
        const data = (await apiClient.listRepos()) as Repo[];
        if (!cancelled) {
          setRepos(data);
        }
      } catch {
        if (!cancelled) {
          setRepos([]);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    loadRepos();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="min-h-screen bg-[color:var(--bg-base)]">
      <header className="sticky top-0 z-50 flex h-16 items-center justify-between border-b border-[color:var(--border)] bg-white/85 px-6 backdrop-blur">
        <div className="flex items-center gap-2 text-lg font-extrabold">
          <span>🐙</span>
          <span>OpenHub</span>
        </div>
        <div className="flex items-center gap-3">
          <Link href="/new" className="rounded-md px-3 py-1.5 text-sm hover:bg-[color:var(--bg-muted)]">
            ＋ 新規
          </Link>
          <Link
            href="/dashboard"
            className="rounded-md bg-[color:var(--primary-light)] px-3 py-1.5 text-sm font-medium text-[color:var(--primary)]"
          >
            ダッシュボード
          </Link>
        </div>
      </header>

      <div className="flex min-h-[calc(100vh-64px)]">
        <aside className="w-60 border-r border-[color:var(--border)] bg-[color:var(--bg-sidebar)] py-5">
          <div className="mb-6 px-4">
            <div className="mb-2 text-xs font-semibold uppercase text-[color:var(--text-muted)]">
              ナビゲーション
            </div>
            <Link
              href="/dashboard"
              className="flex items-center gap-2 rounded-md bg-[color:var(--primary-light)] px-3 py-2 text-sm font-medium text-[color:var(--primary)]"
            >
              📊 ダッシュボード
            </Link>
            <Link
              href="/dashboard"
              className="flex items-center justify-between rounded-md px-3 py-2 text-sm text-[color:var(--text-secondary)] hover:bg-[color:var(--bg-muted)]"
            >
              <span>📁 リポジトリ</span>
              {!loading && (
                <span className="rounded-full bg-white px-1.5 py-0.5 text-xs">{repos.length}</span>
              )}
            </Link>
            <Link
              href="/new"
              className="flex items-center gap-2 rounded-md px-3 py-2 text-sm text-[color:var(--text-secondary)] hover:bg-[color:var(--bg-muted)]"
            >
              ＋ 新規作成
            </Link>
          </div>
        </aside>

        <main className="max-w-[1280px] flex-1 p-8">
          <div className="mb-6 flex items-center justify-between">
            <div>
              <h1 className="m-0 text-2xl font-semibold">リポジトリ</h1>
              {!loading && (
                <p className="mt-1 text-sm text-[color:var(--text-muted)]">
                  {repos.length}個のリポジトリがあります
                </p>
              )}
            </div>
            <Link
              href="/new"
              className="rounded-md bg-[color:var(--primary)] px-4 py-2 text-sm text-white hover:bg-[color:var(--primary-hover)]"
            >
              ＋ 新規作成
            </Link>
          </div>

          {loading ? (
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              {Array.from({ length: 4 }).map((_, i) => (
                <RepoCardSkeleton key={i} />
              ))}
            </div>
          ) : repos.length === 0 ? (
            <div className="rounded-lg border border-[color:var(--border)] bg-white p-8 text-center text-[color:var(--text-secondary)]">
              <p className="mb-4">No repositories yet — create one</p>
              <Link href="/new" className="text-[color:var(--primary)] hover:underline">
                最初のリポジトリを作成する →
              </Link>
            </div>
          ) : (
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              {repos.map((repo) => (
                <Link
                  key={`${repo.owner}/${repo.name}`}
                  href={`/${repo.owner}/${repo.name}`}
                  className="block rounded-lg border border-[color:var(--border)] bg-white p-4 transition-colors hover:border-[color:var(--primary)]"
                >
                  <div className="mb-2 flex items-start justify-between">
                    <h3 className="m-0 text-base font-semibold text-[color:var(--primary)]">
                      {repo.name}
                    </h3>
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs font-semibold ${visibilityBadgeClass(repo.visibility)}`}
                    >
                      {visibilityLabel(repo.visibility)}
                    </span>
                  </div>
                  <p className="my-1 text-xs text-[color:var(--text-secondary)]">
                    Default branch: {repo.default_branch}
                  </p>
                  <p className="text-xs text-[color:var(--text-muted)]">
                    Created: {formatCreated(repo.created_at)}
                  </p>
                </Link>
              ))}
            </div>
          )}
        </main>
      </div>
    </div>
  );
}
