import Link from "next/link";

const features = [
  { icon: "📦", title: "Gitリポジトリ管理", desc: "無制限のプライベート/パブリックリポジトリ。LFS対応、ブランチ保護、コードオーナー、署名コミット検証まで完備。" },
  { icon: "⚡", title: "CI/CDパイプライン", desc: "GitHub Actions互換のワークフロー。既存の `.github/workflows/*.yml` をそのまま実行可能。" },
  { icon: "🔀", title: "プルリクエスト", desc: "レビュー、インラインコメント、必須レビュアー、マージキュー、自動マージなど高度なPRワークフロー。" },
  { icon: "🐛", title: "Issue & プロジェクト", desc: "カンバンボード、マイルストーン、ラベル、サブタスク。アジャイル開発を強力にサポート。" },
  { icon: "📚", title: "パッケージレジストリ", desc: "npm、Docker、Maven、PyPI、NuGetなど主要パッケージ形式をネイティブサポート。" },
  { icon: "🔐", title: "SSO & 監査ログ", desc: "SAML/OIDC、LDAP、2FA、SCIM自動プロビジョニング、完全な監査ログでエンタープライズ対応。" },
];

const stats = [
  { num: "48k+", label: "GitHub Stars" },
  { num: "12k+", label: "アクティブ組織" },
  { num: "100%", label: "オープンソース" },
  { num: "MIT", label: "ライセンス" },
];

const compats = [
  { ico: "🐙", label: "GitHub CLI" },
  { ico: "⚙️", label: "Actions" },
  { ico: "🤖", label: "Dependabot" },
  { ico: "📊", label: "REST/GraphQL API" },
  { ico: "🔗", label: "Webhooks" },
  { ico: "📱", label: "VS Code拡張" },
];

export default function LandingPage() {
  return (
    <div className="min-h-screen bg-[#0d1117] text-[#e6edf3] font-sans">
      {/* Header */}
      <header className="sticky top-0 z-50 flex items-center justify-between px-8 py-4 border-b border-[#30363d] bg-[rgba(13,17,23,0.85)] backdrop-blur-md">
        <Link href="/01-landing" className="flex items-center gap-2.5 font-bold text-lg text-[#e6edf3] no-underline">
          <span className="w-8 h-8 rounded-lg bg-gradient-to-br from-[#1f6feb] to-[#8957e5] flex items-center justify-center text-lg">⑂</span>
          <span>OpenForge</span>
        </Link>
        <nav className="hidden md:flex gap-6 items-center">
          <Link href="/01-landing" className="text-sm text-[#e6edf3] hover:text-[#58a6ff] no-underline">機能</Link>
          <Link href="/01-landing" className="text-sm text-[#e6edf3] hover:text-[#58a6ff] no-underline">ドキュメント</Link>
          <Link href="/01-landing" className="text-sm text-[#e6edf3] hover:text-[#58a6ff] no-underline">価格</Link>
          <Link href="/01-landing" className="text-sm text-[#e6edf3] hover:text-[#58a6ff] no-underline">企業向け</Link>
        </nav>
        <div className="flex gap-3">
          <Link href="/02-signin" className="px-4 py-2 rounded-lg text-sm text-[#e6edf3] hover:bg-[#21262d] no-underline">サインイン</Link>
          <Link href="/03-signup" className="px-4 py-2 rounded-lg text-sm font-medium bg-[#1f6feb] text-white hover:bg-[#1f6febdd] no-underline">無料で始める</Link>
        </div>
      </header>

      {/* Hero */}
      <section className="text-center px-6 pt-24 pb-28 border-b border-[#30363d] bg-[radial-gradient(ellipse_at_top,#1f6feb33_0%,#0d1117_60%)]">
        <span className="inline-block px-3.5 py-1.5 border border-[#30363d] rounded-full text-xs text-[#7d8590] mb-6 bg-[#161b22]">🎉 v2.4 リリース — GitHub Actions互換ワークフローエンジン搭載</span>
        <h1 className="text-5xl md:text-6xl leading-tight font-extrabold tracking-tight mb-5">
          Gitプラットフォームを、<br />
          <span className="bg-gradient-to-r from-[#1f6feb] via-[#8957e5] to-[#ec4899] bg-clip-text text-transparent">あなたのインフラへ。</span>
        </h1>
        <p className="text-lg md:text-xl text-[#7d8590] max-w-2xl mx-auto mb-10 leading-relaxed">
          OpenForgeはセルフホスト型のGitHub互換プラットフォーム。完全オープンソース、API互換、データ主権を保ちながら、開発チームに最高のコラボレーション体験を提供します。
        </p>
        <div className="flex gap-4 justify-center flex-wrap">
          <Link href="/03-signup" className="px-6 py-3 rounded-lg bg-[#1f6feb] text-white font-medium hover:bg-[#1f6febdd] no-underline">🚀 無料でホスティング開始</Link>
          <Link href="/01-landing" className="px-6 py-3 rounded-lg border border-[#30363d] text-[#e6edf3] font-medium hover:bg-[#21262d] no-underline">📖 ドキュメントを見る</Link>
        </div>
        <div className="flex gap-12 justify-center mt-14 flex-wrap">
          {stats.map((s) => (
            <div key={s.label} className="text-center">
              <div className="text-4xl font-bold text-[#58a6ff]">{s.num}</div>
              <div className="text-sm text-[#7d8590] mt-1">{s.label}</div>
            </div>
          ))}
        </div>
      </section>

      {/* Features */}
      <section className="px-6 py-24 border-b border-[#21262d]">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-14">
            <h2 className="text-4xl font-bold tracking-tight mb-4">開発に必要な、すべてが揃う。</h2>
            <p className="text-[#7d8590] text-lg max-w-xl mx-auto">コード管理からCI/CD、パッケージレジストリ、Issue追跡まで。エンタープライズグレードの機能をオールインワンで。</p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {features.map((f) => (
              <div key={f.title} className="bg-[#161b22] border border-[#30363d] rounded-xl p-7 hover:border-[#58a6ff] transition-colors">
                <div className="w-12 h-12 rounded-[10px] bg-gradient-to-br from-[#1f6feb] to-[#8957e5] flex items-center justify-center text-2xl mb-4">{f.icon}</div>
                <h3 className="text-lg mb-2 font-semibold">{f.title}</h3>
                <p className="text-sm text-[#7d8590] leading-relaxed">{f.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Deploy */}
      <section className="px-6 py-24 border-b border-[#21262d] bg-[#0a0d12]">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-14">
            <h2 className="text-4xl font-bold tracking-tight mb-4">5分でデプロイ。</h2>
            <p className="text-[#7d8590] text-lg">DockerでもKubernetes (Helm) でも、お好みの方法でセルフホスト。</p>
          </div>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="bg-[#010409] border border-[#30363d] rounded-xl overflow-hidden">
              <div className="px-4 py-3 bg-[#161b22] border-b border-[#30363d] flex items-center justify-between text-xs">
                <span className="font-semibold">🐳 Docker Compose</span>
                <span className="text-[#7d8590]">docker-compose.yml</span>
              </div>
              <pre className="m-0 p-5 font-mono text-xs leading-relaxed text-[#c9d1d9] overflow-x-auto">
<span className="text-[#8b949e]"># 単一コマンドで起動</span>{"\n"}
<span className="text-[#ff7b72]">version:</span> <span className="text-[#a5d6ff]">{`'3.8'`}</span>{"\n"}
<span className="text-[#ff7b72]">services:</span>{"\n"}
{"  "}<span className="text-[#ff7b72]">openforge:</span>{"\n"}
{"    "}<span className="text-[#ff7b72]">image:</span> openforge/server:latest{"\n"}
{"    "}<span className="text-[#ff7b72]">ports:</span>{"\n"}
{"      - "}<span className="text-[#a5d6ff]">{`"3000:3000"`}</span>{"\n"}
{"      - "}<span className="text-[#a5d6ff]">{`"2222:22"`}</span>{"\n"}
{"    "}<span className="text-[#ff7b72]">volumes:</span>{"\n"}
{"      - ./data:/var/lib/openforge\n"}
{"    "}<span className="text-[#ff7b72]">environment:</span>{"\n"}
{"      OPENFORGE_DOMAIN: git.example.com"}
              </pre>
            </div>
            <div className="bg-[#010409] border border-[#30363d] rounded-xl overflow-hidden">
              <div className="px-4 py-3 bg-[#161b22] border-b border-[#30363d] flex items-center justify-between text-xs">
                <span className="font-semibold">☸️ Helm Chart</span>
                <span className="text-[#7d8590]">install.sh</span>
              </div>
              <pre className="m-0 p-5 font-mono text-xs leading-relaxed text-[#c9d1d9] overflow-x-auto">
<span className="text-[#8b949e]"># Helmリポジトリを追加</span>{"\n"}
{"$ helm repo add openforge \\\n"}
{"    "}<span className="text-[#a5d6ff]">https://charts.openforge.io</span>{"\n\n"}
<span className="text-[#8b949e]"># Kubernetesクラスタにインストール</span>{"\n"}
{"$ helm install my-forge openforge/openforge \\\n"}
{"    --namespace forge \\\n"}
{"    --create-namespace \\\n"}
{"    --set domain="}<span className="text-[#a5d6ff]">git.example.com</span>{" \\\n"}
{"    --set persistence.size="}<span className="text-[#a5d6ff]">100Gi</span>{"\n\n"}
<span className="text-[#8b949e]"># 完了！</span>{"\n"}
{"✓ Deployed in 47s"}
              </pre>
            </div>
          </div>
        </div>
      </section>

      {/* Compat */}
      <section className="px-6 py-24 border-b border-[#21262d]">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-14">
            <h2 className="text-4xl font-bold tracking-tight mb-4">GitHubエコシステムと完全互換。</h2>
            <p className="text-[#7d8590] text-lg">既存のツールチェーンをそのまま活用。移行コストを最小化します。</p>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4 mt-8">
            {compats.map((c) => (
              <div key={c.label} className="bg-[#161b22] border border-[#30363d] rounded-[10px] py-6 px-3 text-center text-[#7d8590] text-xs">
                <span className="text-3xl block mb-2">{c.ico}</span>
                {c.label}
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Final CTA */}
      <section className="text-center px-6 py-20 bg-gradient-to-br from-[#1f6feb] to-[#8957e5]">
        <div className="max-w-6xl mx-auto">
          <h2 className="text-4xl md:text-5xl font-bold mb-4 text-white">あなたのコードを、あなたのサーバーへ。</h2>
          <p className="text-lg text-white/85 mb-8">クレジットカード不要。今すぐクラウド版を試すか、自分のインフラへインストール。</p>
          <div className="flex gap-4 justify-center flex-wrap">
            <Link href="/03-signup" className="px-6 py-3 rounded-lg bg-white text-[#1f6feb] font-medium no-underline hover:bg-white/90">無料アカウント作成</Link>
            <Link href="/02-signin" className="px-6 py-3 rounded-lg border border-white/50 text-white font-medium no-underline hover:bg-white/10">既存アカウントでサインイン</Link>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-[#0a0d12] px-6 pt-16 pb-8 text-[#7d8590] text-sm">
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-10 max-w-6xl mx-auto mb-12">
          <div className="col-span-2 md:col-span-3 lg:col-span-1">
            <Link href="/01-landing" className="flex items-center gap-2.5 font-bold text-lg text-[#e6edf3] no-underline mb-4">
              <span className="w-8 h-8 rounded-lg bg-gradient-to-br from-[#1f6feb] to-[#8957e5] flex items-center justify-center text-lg">⑂</span>
              <span>OpenForge</span>
            </Link>
            <p className="max-w-[280px]">セルフホスト可能な、オープンソースのGitプラットフォーム。MITライセンスで提供。</p>
          </div>
          <div>
            <h4 className="text-[#e6edf3] text-sm mb-4">プロダクト</h4>
            <ul className="space-y-2.5">
              <li><Link href="/01-landing" className="text-[#7d8590] no-underline hover:text-[#e6edf3]">機能一覧</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] no-underline hover:text-[#e6edf3]">価格プラン</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] no-underline hover:text-[#e6edf3]">ロードマップ</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] no-underline hover:text-[#e6edf3]">変更履歴</Link></li>
            </ul>
          </div>
          <div>
            <h4 className="text-[#e6edf3] text-sm mb-4">リソース</h4>
            <ul className="space-y-2.5">
              <li><Link href="/01-landing" className="text-[#7d8590] no-underline hover:text-[#e6edf3]">ドキュメント</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] no-underline hover:text-[#e6edf3]">APIリファレンス</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] no-underline hover:text-[#e6edf3]">ブログ</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] no-underline hover:text-[#e6edf3]">コミュニティ</Link></li>
            </ul>
          </div>
          <div>
            <h4 className="text-[#e6edf3] text-sm mb-4">会社情報</h4>
            <ul className="space-y-2.5">
              <li><Link href="/01-landing" className="text-[#7d8590] no-underline hover:text-[#e6edf3]">概要</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] no-underline hover:text-[#e6edf3]">採用情報</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] no-underline hover:text-[#e6edf3]">お問い合わせ</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] no-underline hover:text-[#e6edf3]">セキュリティ</Link></li>
            </ul>
          </div>
          <div>
            <h4 className="text-[#e6edf3] text-sm mb-4">法的事項</h4>
            <ul className="space-y-2.5">
              <li><Link href="/01-landing" className="text-[#7d8590] no-underline hover:text-[#e6edf3]">利用規約</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] no-underline hover:text-[#e6edf3]">プライバシー</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] no-underline hover:text-[#e6edf3]">ライセンス</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] no-underline hover:text-[#e6edf3]">GDPR</Link></li>
            </ul>
          </div>
        </div>
        <div className="border-t border-[#21262d] pt-6 text-center max-w-6xl mx-auto">
          © 2025 OpenForge Project · Released under the MIT License · Made with ⑂ by contributors worldwide
        </div>
      </footer>
    </div>
  );
}
