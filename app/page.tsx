import Link from "next/link";

type Screen = {
  id: string;
  name: string;
  desc: string;
  href: string;
};

const screens: Screen[] = [
  {
    id: "01-landing",
    name: "ランディングページ",
    desc: "セルフホスト型GitHub互換プラットフォームの紹介ページ。主要機能、デプロイ方法、GitHubエコシステム互換性をアピール...",
    href: "/landing",
  },
  {
    id: "02-signin",
    name: "サインイン画面",
    desc: "ユーザー名/メールとパスワードでログイン。OAuth Appsログインも可能...",
    href: "/signin",
  },
  {
    id: "04-dashboard",
    name: "ダッシュボード",
    desc: "ログイン後のホーム。アクティビティフィード、自分のリポジトリ、最近のIssue/PR、組織を表示...",
    href: "/dashboard",
  },
  {
    id: "07-repo-detail",
    name: "リポジトリ詳細（コードブラウザ）",
    desc: "リポジトリのコード閲覧画面。ファイルツリー、READMEレンダリング、ブランチ切替、クローンURL表示...",
    href: "/repo",
  },
  {
    id: "13-pr-list",
    name: "Pull Request一覧・詳細",
    desc: "PRの一覧と詳細。レビュー、diff、コメント、マージ操作...",
    href: "/pulls",
  },
];

export default function Page() {
  return (
    <main className="min-h-screen bg-[color:var(--bg-base)] p-8">
      <div className="bg-[color:var(--bg-elevated)] p-8 rounded-xl shadow-[var(--shadow-md)] mb-8">
        <h1 className="mb-2 text-[color:var(--text-primary)]">
          🎨 <span className="text-gradient">オープンソースGitHub</span>
        </h1>
        <p className="text-[color:var(--text-secondary)] flex items-center gap-3 flex-wrap">
          インタラクティブプロトタイプ
          <span className="inline-block px-3 py-1 rounded-full text-xs font-semibold bg-[color:var(--success-light)] text-[color:var(--success)]">
            desktop
          </span>
          <span>{screens.length} 画面</span>
        </p>
      </div>

      <ul className="grid gap-4">
        {screens.map((s) => (
          <li
            key={s.id}
            className="bg-[color:var(--bg-elevated)] rounded-lg shadow-[var(--shadow-sm)] hover:shadow-[var(--shadow-md)] transition"
          >
            <Link
              href={s.href}
              className="flex items-center gap-4 px-6 py-4 no-underline text-inherit hover:bg-[color:var(--primary-light)] rounded-lg"
            >
              <span className="mono text-xs text-[color:var(--text-muted)] min-w-[80px]">
                {s.id}
              </span>
              <span className="font-semibold text-[color:var(--text-primary)] min-w-[200px]">
                {s.name}
              </span>
              <span className="text-[color:var(--text-secondary)] text-sm">
                {s.desc}
              </span>
            </Link>
          </li>
        ))}
      </ul>
    </main>
  );
}
