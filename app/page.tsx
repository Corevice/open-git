import Link from "next/link";

const screens = [
  { id: "01-landing", name: "ランディングページ", desc: "セルフホスト型GitHub互換プラットフォームの紹介ページ。主要機能、デプロイ方法、GitHubエコシステム互換性をアピール..." },
  { id: "02-signin", name: "サインイン画面", desc: "ユーザー名/メールとパスワードでログイン。OAuth Appsログインも可能..." },
  { id: "03-signup", name: "サインアップ画面", desc: "新規ユーザー登録フォーム。ユーザー名、メール、パスワードを入力..." },
  { id: "04-dashboard", name: "ダッシュボード", desc: "ログイン後のホーム。アクティビティフィード、自分のリポジトリ、最近のIssue/PR、組織を表示..." },
  { id: "05-repo-list", name: "リポジトリ一覧", desc: "ユーザー/組織のリポジトリ一覧。フィルタ・ソート・検索が可能..." },
  { id: "06-repo-create", name: "リポジトリ作成画面", desc: "新規リポジトリ作成フォーム。名前空間、公開範囲、README初期化等を設定..." },
  { id: "07-repo-detail", name: "リポジトリ詳細（コードブラウザ）", desc: "リポジトリのコード閲覧画面。ファイルツリー、READMEレンダリング、ブランチ切替、クローンURL表示..." },
  { id: "08-file-viewer", name: "ファイル表示・編集画面", desc: "個別ファイルのコード表示。シンタックスハイライト、行番号、編集モード切替対応..." },
  { id: "09-commit-history", name: "コミット履歴・Diff表示", desc: "コミット一覧と選択時のdiff表示。比較ビュー対応..." },
  { id: "10-actions-list", name: "Actionsワークフロー一覧", desc: "GitHub Actions互換のワークフロー実行履歴。ジョブステータス、ログ閲覧可能..." },
  { id: "11-repo-settings", name: "リポジトリ設定", desc: "リポジトリのオプション設定。コラボレーター、Webhook、ブランチ保護、危険ゾーン..." },
  { id: "12-issue-list", name: "Issue一覧・詳細", desc: "Issueの一覧と選択時の詳細パネル。ラベル、マイルストーン、コメント、ステータス管理..." },
  { id: "13-pr-list", name: "Pull Request一覧・詳細", desc: "PRの一覧と詳細。レビュー、diff、コメント、マージ操作..." },
  { id: "14-import-wizard", name: "GitHubインポートウィザード", desc: "既存GitHubリポジトリの移行ウィザード。URL入力、認証、移行範囲選択、進捗表示..." },
  { id: "15-settings", name: "ユーザー設定（PAT/OAuth）", desc: "プロファイル、Personal Access Token、OAuth Apps、SSHキー管理..." },
];

export default function Page() {
  return (
    <div className="min-h-screen p-8 bg-[color:var(--bg-base)]">
      <div className="bg-white p-8 rounded-xl shadow-md mb-8">
        <h1 className="text-3xl font-bold mb-2 text-[color:var(--text-primary)]">
          🎨 オープンソースGitHub
        </h1>
        <p className="text-[color:var(--text-secondary)] flex items-center gap-3 flex-wrap">
          インタラクティブプロトタイプ
          <span className="inline-block px-3 py-1 rounded-full text-xs font-semibold bg-[color:var(--success-light)] text-[color:var(--success)]">
            desktop
          </span>
          15 画面
        </p>
      </div>
      <ul className="grid gap-4">
        {screens.map((s) => (
          <li key={s.id} className="bg-white rounded-lg shadow-sm">
            <Link
              href={`/${s.id}`}
              className="flex items-center gap-4 px-6 py-4 hover:bg-[color:var(--primary-light)] rounded-lg transition-colors"
            >
              <span className="font-mono text-xs text-[color:var(--text-muted)] min-w-[80px]">
                {s.id}
              </span>
              <span className="font-semibold text-[color:var(--text-primary)] min-w-[200px]">
                {s.name}
              </span>
              <span className="text-sm text-[color:var(--text-secondary)]">
                {s.desc}
              </span>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
