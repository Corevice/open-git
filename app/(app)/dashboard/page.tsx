import Link from "next/link";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";

interface UserRepo {
  id: number;
  name: string;
  full_name: string;
  description: string | null;
  private: boolean;
  visibility?: string;
  language: string | null;
  stargazers_count: number;
  forks_count: number;
  updated_at: string;
  owner: { login: string };
}

const LANG_COLORS: Record<string, string> = {
  JavaScript: "#f1e05a",
  TypeScript: "#3178c6",
  Python: "#3572A5",
  Go: "#00ADD8",
  Rust: "#dea584",
  CSS: "#563d7c",
  Shell: "#89e051",
};

async function apiGet<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { Accept: "application/vnd.github+json" },
    cache: "no-store",
  });
  if (!res.ok) return [] as T;
  return res.json() as Promise<T>;
}

function visibilityBadgeClass(repo: UserRepo): string {
  const v = (repo.visibility ?? (repo.private ? "private" : "public")).toLowerCase();
  if (v === "public") return "bg-[color:var(--success-light)] text-[color:var(--success)]";
  if (v === "private") return "bg-[color:var(--warning-light)] text-[color:var(--warning)]";
  return "bg-[color:var(--info-light)] text-[color:var(--info)]";
}

function visibilityLabel(repo: UserRepo): string {
  if (repo.visibility) {
    return repo.visibility.charAt(0).toUpperCase() + repo.visibility.slice(1);
  }
  return repo.private ? "Private" : "Public";
}

function formatUpdated(iso: string): string {
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return iso;
  const seconds = Math.floor((Date.now() - then) / 1000);
  if (seconds < 3600) return `${Math.max(1, Math.floor(seconds / 60))}分前`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}時間前`;
  if (seconds < 86400 * 30) return `${Math.floor(seconds / 86400)}日前`;
  return new Date(iso).toLocaleDateString("ja-JP");
}

function formatStars(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(1).replace(/\.0$/, "")}k`;
  return String(n);
}

export default async function DashboardPage() {
  const repos = await apiGet<UserRepo[]>("/user/repos?sort=updated&per_page=100");

  return (
    <div className="min-h-screen bg-[color:var(--bg-base)]">
      <header className="h-16 sticky top-0 z-50 flex items-center justify-between px-6 border-b border-[color:var(--border)] bg-white/85 backdrop-blur">
        <div className="flex items-center gap-2 text-lg font-extrabold">
          <span>🐙</span>
          <span>OpenHub</span>
        </div>
        <div className="flex items-center gap-3">
          <Link href="/new" className="px-3 py-1.5 text-sm rounded-md hover:bg-[color:var(--bg-muted)]">
            ＋ 新規
          </Link>
          <Link
            href="/dashboard"
            className="px-3 py-1.5 text-sm rounded-md bg-[color:var(--primary-light)] text-[color:var(--primary)] font-medium"
          >
            ダッシュボード
          </Link>
        </div>
      </header>

      <div className="flex min-h-[calc(100vh-64px)]">
        <aside className="w-60 bg-[color:var(--bg-sidebar)] border-r border-[color:var(--border)] py-5">
          <div className="px-4 mb-6">
            <div className="text-xs font-semibold text-[color:var(--text-muted)] uppercase mb-2">
              ナビゲーション
            </div>
            <Link
              href="/dashboard"
              className="flex items-center gap-2 px-3 py-2 rounded-md text-sm bg-[color:var(--primary-light)] text-[color:var(--primary)] font-medium"
            >
              📊 ダッシュボード
            </Link>
            <Link
              href="/dashboard"
              className="flex items-center justify-between px-3 py-2 rounded-md text-sm text-[color:var(--text-secondary)] hover:bg-[color:var(--bg-muted)]"
            >
              <span>📁 リポジトリ</span>
              <span className="text-xs px-1.5 py-0.5 rounded-full bg-white">{repos.length}</span>
            </Link>
            <Link
              href="/new"
              className="flex items-center gap-2 px-3 py-2 rounded-md text-sm text-[color:var(--text-secondary)] hover:bg-[color:var(--bg-muted)]"
            >
              ＋ 新規作成
            </Link>
          </div>
        </aside>

        <main className="flex-1 p-8 max-w-[1280px]">
          <div className="flex justify-between items-center mb-6">
            <div>
              <h1 className="text-2xl font-semibold m-0">リポジトリ</h1>
              <p className="text-[color:var(--text-muted)] text-sm mt-1">
                {repos.length}個のリポジトリがあります
              </p>
            </div>
            <Link
              href="/new"
              className="px-4 py-2 text-sm rounded-md bg-[color:var(--primary)] text-white hover:bg-[color:var(--primary-hover)]"
            >
              ＋ 新規作成
            </Link>
          </div>

          {repos.length === 0 ? (
            <div className="bg-white border border-[color:var(--border)] rounded-lg p-8 text-center text-[color:var(--text-secondary)]">
              <p className="mb-4">リポジトリがありません。</p>
              <Link href="/new" className="text-[color:var(--primary)] hover:underline">
                最初のリポジトリを作成する →
              </Link>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {repos.map((r) => {
                const [ownerLogin] = r.full_name.split("/");
                const langColor = LANG_COLORS[r.language ?? ""] ?? "#8b949e";
                return (
                  <Link
                    key={r.id}
                    href={`/${ownerLogin}/${r.name}`}
                    className="block bg-white border border-[color:var(--border)] rounded-lg p-4 transition-colors hover:border-[color:var(--primary)]"
                  >
                    <div className="flex justify-between items-start mb-2">
                      <h3 className="text-base font-semibold text-[color:var(--primary)] m-0">
                        {r.private ? "🔒" : "📘"} {r.name}
                      </h3>
                      <span
                        className={`text-xs font-semibold px-2 py-0.5 rounded-full ${visibilityBadgeClass(r)}`}
                      >
                        {visibilityLabel(r)}
                      </span>
                    </div>
                    <p className="text-[13px] text-[color:var(--text-secondary)] my-2 leading-relaxed line-clamp-2">
                      {r.description || "説明なし"}
                    </p>
                    <div className="flex gap-4 text-xs text-[color:var(--text-secondary)] items-center flex-wrap">
                      {r.language && (
                        <span className="flex items-center gap-1">
                          <span
                            className="inline-block w-2.5 h-2.5 rounded-full"
                            style={{ backgroundColor: langColor }}
                          />
                          {r.language}
                        </span>
                      )}
                      <span>⭐ {formatStars(r.stargazers_count)}</span>
                      <span>🍴 {r.forks_count}</span>
                      <span>更新: {formatUpdated(r.updated_at)}</span>
                    </div>
                  </Link>
                );
              })}
            </div>
          )}
        </main>
      </div>
    </div>
  );
}
