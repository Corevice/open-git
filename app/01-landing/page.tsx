import Link from "next/link";

const features = [
  { icon: "📦", title: "Gitリポジトリ管理", desc: "無制限のプライベート/パブリックリポジトリ。LFS対応、ブランチ保護、コードオーナー、署名コミット検証まで完備。" },
  { icon: "⚡", title: "CI/CDパイプライン", desc: "GitHub Actions互換のワークフロー。既存の .github/workflows/*.yml をそのまま実行可能。" },
  { icon: "🔀", title: "プルリクエスト", desc: "レビュー、インラインコメント、必須レビュアー、マージキュー、自動マージなど高度なPRワークフロー。" },
  { icon: "🐛", title: "Issue & プロジェクト", desc: "カンバンボード、マイルストーン、ラベル、サブタスク。アジャイル開発を強力にサポート。" },
  { icon: "📚", title: "パッケージレジストリ", desc: "npm、Docker、Maven、PyPI、NuGetなど主要パッケージ形式をネイティブサポート。" },
];

const stats = [
  { num: "48k+", label: "GitHub Stars" },
  { num: "12k+", label: "アクティブ組織" },
  { num: "100%", label: "オープンソース" },
  { num: "MIT", label: "ライセンス" },
];

const compat = [
  { ico: "🐙", name: "GitHub CLI" },
  { ico: "⚙️", name: "Actions" },
  { ico: "🤖", name: "Dependabot" },
  { ico: "📊", name: "REST/GraphQL API" },
  { ico: "🔗", name: "Webhooks" },
  { ico: "📱", name: "VS Code拡張" },
];

export default function LandingPage() {
  return (
    <div className="min-h-screen bg-[#0d1117] text-[#e6edf3] font-sans">
      {/* Header */}
      <header className="flex items-center justify-between px-8 py-4 border-b border-[#30363d] bg-[rgba(13,17,23,0.85)] backdrop-blur-md sticky top-0 z-50">
        <Link href="/01-landing" className="flex items-center gap-2.5 font-bold text-lg text-[#e6edf3]">
          <span className="w-8 h-8 rounded-lg bg-gradient-to-br from-[#1f6feb] to-[#8957e5] flex items-center justify-center text-lg">⑂</span>
          <span>OpenForge</span>
        </Link>
        <nav className="hidden md:flex gap-6 items-center">
          <Link href="/01-landing" className="text-[#e6edf3] text-sm hover:text-[#58a6ff]">機能</Link>
          <Link href="/01-landing" className="text-[#e6edf3] text-sm hover:text-[#58a6ff]">ドキュメント</Link>
          <Link href="/01-landing" className="text-[#e6edf3] text-sm hover:text-[#58a6ff]">価格</Link>
          <Link href="/01-landing" className="text-[#e6edf3] text-sm hover:text-[#58a6ff]">企業向け</Link>
        </nav>
        <div className="flex gap-3">
          <Link href="/02-signin" className="px-4 py-2 text-sm text-[#e6edf3] hover:bg-[#21262d] rounded-md">サインイン</Link>
          <Link href="/03-signup" className="px-4 py-2 text-sm bg-[#238636] hover:bg-[#2ea043] text-white rounded-md font-medium">無料で始める</Link>
        </div>
      </header>

      {/* Hero */}
      <section className="px-6 pt-24 pb-32 text-center border-b border-[#30363d] bg-[radial-gradient(ellipse_at_top,#1f6feb33_0%,#0d1117_60%)]">
        <span className="inline-block px-3.5 py-1.5 border border-[#30363d] rounded-full text-[13px] text-[#7d8590] mb-6 bg-[#161b22]">
          🎉 v2.4 リリース — GitHub Actions互換ワークフローエンジン搭載
        </span>
        <h1 className="text-5xl md:text-6xl font-extrabold leading-[1.1] mb-5 tracking-tight">
          Gitプラットフォームを、<br />
          <span className="bg-gradient-to-r from-[#1f6feb] via-[#8957e5] to-[#ec4899] bg-clip-text text-transparent">あなたのインフラへ。</span>
        </h1>
        <p className="text-lg text-[#7d8590] max-w-[720px] mx-auto mb-10 leading-relaxed">
          OpenForgeはセルフホスト型のGitHub互換プラットフォーム。完全オープンソース、API互換、データ主権を保ちながら、開発チームに最高のコラボレーション体験を提供します。
        </p>
        <div className="flex gap-4 justify-center flex-wrap">
          <Link href="/03-signup" className="px-6 py-3 bg-[#238636] hover:bg-[#2ea043] text-white rounded-md font-medium">🚀 無料でホスティング開始</Link>
          <Link href="/01-landing" className="px-6 py-3 border border-[#30363d] hover:bg-[#21262d] text-[#e6edf3] rounded-md font-medium">📖 ドキュメントを見る</Link>
        </div>
        <div className="flex gap-8 md:gap-12 justify-center mt-14 flex-wrap">
          {stats.map((s) => (
            <div key={s.label} className="text-center">
              <div className="text-4xl font-bold text-[#58a6ff]">{s.num}</div>
              <div className="text-[#7d8590] text-sm mt-1">{s.label}</div>
            </div>
          ))}
        </div>
      </section>

      {/* Features */}
      <section className="px-6 py-24 border-b border-[#21262d]">
        <div className="max-w-[1200px] mx-auto">
          <div className="text-center mb-14">
            <h2 className="text-4xl font-bold mb-4 tracking-tight">開発に必要な、すべてが揃う。</h2>
            <p className="text-[#7d8590] text-lg max-w-[640px] mx-auto">コード管理からCI/CD、パッケージレジストリ、Issue追跡まで。エンタープライズグレードの機能をオールインワンで。</p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {features.map((f) => (
              <div key={f.title} className="bg-[#161b22] border border-[#30363d] rounded-xl p-7 hover:border-[#58a6ff] transition-colors">
                <div className="w-12 h-12 rounded-lg bg-gradient-to-br from-[#1f6feb] to-[#8957e5] flex items-center justify-center text-2xl mb-4">{f.icon}</div>
                <h3 className="text-lg font-semibold mb-2">{f.title}</h3>
                <p className="text-[#7d8590] text-sm leading-relaxed">{f.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Deploy */}
      <section className="px-6 py-24 border-b border-[#21262d] bg-[#0a0d12]">
        <div className="max-w-[1200px] mx-auto">
          <div className="text-center mb-14">
            <h2 className="text-4xl font-bold mb-4 tracking-tight">5分でデプロイ。</h2>
            <p className="text-[#7d8590] text-lg max-w-[640px] mx-auto">DockerでもKubernetes (Helm) でも、お好みの方法でセルフホスト。</p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="bg-[#010409] border border-[#30363d] rounded-xl overflow-hidden">
              <div className="px-4 py-3 bg-[#161b22] border-b border-[#30363d] flex items-center justify-between text-[13px]">
                <span className="font-semibold">🐳 Docker Compose</span>
                <span className="text-[#7d8590]">docker-compose.yml</span>
              </div>
              <pre className="m-0 p-5 font-mono text-[13px] leading-[1.7] text-[#c9d1d9] overflow-x-auto">
<span className="text-[#8b949e]"># 単一コマンドで起動</span>{"\n"}
<span className="text-[#ff7b72]">version:</span> <span className="text-[#a5d6ff]">{"'3.8'"}</span>{"\n"}
<span className="text-[#ff7b72]">services:</span>{"\n"}
{"  "}<span className="text-[#ff7b72]">openforge:</span>{"\n"}
{"    "}<span className="text-[#ff7b72]">image:</span> openforge/server:latest{"\n"}
{"    "}<span className="text-[#ff7b72]">ports:</span>{"\n"}
{"      - "}<span className="text-[#a5d6ff]">{'"3000:3000"'}</span>{"\n"}
{"      - "}<span className="text-[#a5d6ff]">{'"2222:22"'}</span>{"\n"}
{"    "}<span className="text-[#ff7b72]">volumes:</span>{"\n"}
{"      - ./data:/var/lib/openforge\n"}
{"    "}<span className="text-[#ff7b72]">environment:</span>{"\n"}
{"      OPENFORGE_DOMAIN: git.example.com"}
              </pre>
            </div>
            <div className="bg-[#010409] border border-[#30363d] rounded-xl overflow-hidden">
              <div className="px-4 py-3 bg-[#161b22] border-b border-[#30363d] flex items-center justify-between text-[13px]">
                <span className="font-semibold">☸️ Helm Chart</span>
                <span className="text-[#7d8590]">install.sh</span>
              </div>
              <pre className="m-0 p-5 font-mono text-[13px] leading-[1.7] text-[#c9d1d9] overflow-x-auto">
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
        <div className="max-w-[1200px] mx-auto">
          <div className="text-center mb-14">
            <h2 className="text-4xl font-bold mb-4 tracking-tight">GitHubエコシステムと完全互換。</h2>
            <p className="text-[#7d8590] text-lg max-w-[640px] mx-auto">既存のツールチェーンをそのまま活用。移行コストを最小化します。</p>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-6 gap-4">
            {compat.map((c) => (
              <div key={c.name} className="bg-[#161b22] border border-[#30363d] rounded-lg px-3 py-6 text-center text-[#7d8590] text-[13px]">
                <span className="text-3xl block mb-2">{c.ico}</span>
                {c.name}
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Final CTA */}
      <section className="px-6 py-20 text-center bg-gradient-to-br from-[#1f6feb] to-[#8957e5]">
        <div className="max-w-[1200px] mx-auto">
          <h2 className="text-4xl font-bold mb-4 text-white">あなたのコードを、あなたのサーバーへ。</h2>
          <p className="text-white/85 text-lg mb-8">クレジットカード不要。今すぐクラウド版を試すか、自分のインフラへインストール。</p>
          <div className="flex gap-4 justify-center flex-wrap">
            <Link href="/03-signup" className="px-6 py-3 bg-white text-[#1f6feb] rounded-md font-medium hover:bg-gray-100">無料アカウント作成</Link>
            <Link href="/02-signin" className="px-6 py-3 border border-white/50 text-white rounded-md font-medium hover:bg-white/10">既存アカウントでサインイン</Link>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-[#0a0d12] px-6 pt-16 pb-8 text-[#7d8590] text-sm">
        <div className="grid grid-cols-2 md:grid-cols-5 gap-10 max-w-[1200px] mx-auto mb-12">
          <div className="col-span-2">
            <Link href="/01-landing" className="flex items-center gap-2.5 font-bold text-lg text-[#e6edf3] mb-4">
              <span className="w-8 h-8 rounded-lg bg-gradient-to-br from-[#1f6feb] to-[#8957e5] flex items-center justify-center text-lg">⑂</span>
              <span>OpenForge</span>
            </Link>
            <p className="max-w-[280px]">セルフホスト可能な、オープンソースのGitプラットフォーム。MITライセンスで提供。</p>
          </div>
          <div>
            <h4 className="text-[#e6edf3] text-sm mb-4 font-semibold">プロダクト</h4>
            <ul className="space-y-2.5">
              <li><Link href="/01-landing" className="text-[#7d8590] hover:text-[#58a6ff]">機能一覧</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] hover:text-[#58a6ff]">価格プラン</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] hover:text-[#58a6ff]">ロードマップ</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] hover:text-[#58a6ff]">変更履歴</Link></li>
            </ul>
          </div>
          <div>
            <h4 className="text-[#e6edf3] text-sm mb-4 font-semibold">リソース</h4>
            <ul className="space-y-2.5">
              <li><Link href="/01-landing" className="text-[#7d8590] hover:text-[#58a6ff]">ドキュメント</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] hover:text-[#58a6ff]">APIリファレンス</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] hover:text-[#58a6ff]">ブログ</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] hover:text-[#58a6ff]">コミュニティ</Link></li>
            </ul>
          </div>
          <div>
            <h4 className="text-[#e6edf3] text-sm mb-4 font-semibold">会社情報</h4>
            <ul className="space-y-2.5">
              <li><Link href="/01-landing" className="text-[#7d8590] hover:text-[#58a6ff]">概要</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] hover:text-[#58a6ff]">採用情報</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] hover:text-[#58a6ff]">お問い合わせ</Link></li>
              <li><Link href="/01-landing" className="text-[#7d8590] hover:text-[#58a6ff]">セキュリティ</Link></li>
            </ul>
          </div>
        </div>
        <div className="border-t border-[#21262d] pt-6 text-center max-w-[1200px] mx-auto">
          © 2025 OpenForge Project · Released under the MIT License · Made with ⑂ by contributors worldwide
        </div>
      </footer>
    </div>
  );
}
