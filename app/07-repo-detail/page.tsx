import Link from "next/link";
import CloneUrlToggle from "@/components/repo/CloneUrlToggle";

const files = [
  { icon: "📁", name: ".github", msg: "Update workflow permissions", age: "3 days ago" },
  { icon: "📁", name: "src", msg: "Refactor authentication module", age: "5 hours ago" },
  { icon: "📁", name: "tests", msg: "Add integration tests for API", age: "2 days ago" },
  { icon: "📄", name: "README.md", msg: "Update installation guide", age: "3 days ago" },
  { icon: "📄", name: "package.json", msg: "Bump version to 2.4.0", age: "1 day ago" },
];

const contributors = [
  { initials: "OC", bg: "bg-[#f97583]" },
  { initials: "JD", bg: "bg-[#79c0ff]" },
  { initials: "AM", bg: "bg-[#a5d6ff]" },
  { initials: "KS", bg: "bg-[#d2a8ff]" },
  { initials: "RT", bg: "bg-[#ffa657]" },
  { initials: "+19", bg: "bg-[#7ee787]" },
];

const languages = [
  { name: "TypeScript", pct: 65, color: "bg-[#3178c6]" },
  { name: "JavaScript", pct: 22, color: "bg-[#f1e05a]" },
  { name: "CSS", pct: 8, color: "bg-[#563d7c]" },
  { name: "Shell", pct: 5, color: "bg-[#89e051]" },
];

export default function RepoDetailPage() {
  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <header className="h-16 bg-white/85 backdrop-blur border-b border-[color:var(--border)] sticky top-0 z-[100]">
        <div className="max-w-[1280px] mx-auto px-6 flex items-center justify-between h-full">
          <div className="text-lg font-extrabold flex items-center gap-2">
            <span>🐙</span> OSS GitHub
          </div>
          <div className="flex items-center gap-3">
            <Link href="/12-issue-list" className="px-2 py-1 text-sm hover:bg-gray-100 rounded">🔔</Link>
            <Link href="/11-repo-settings" className="px-2 py-1 text-sm hover:bg-gray-100 rounded">⚙️</Link>
            <span className="px-2 py-1 rounded-full text-xs font-medium bg-[color:var(--primary-light)] text-[color:var(--primary)]">octocat</span>
          </div>
        </div>
      </header>

      <div className="bg-white border-b border-[#d0d7de] py-4 sticky top-16 z-10">
        <div className="max-w-[1280px] mx-auto px-6">
          <div className="flex items-center gap-3 flex-wrap">
            <h1 className="text-xl m-0 flex items-center gap-2 flex-wrap">
              📁 <Link href="/07-repo-detail" className="text-[#0969da] no-underline hover:underline">octocat</Link>
              <span>/</span>
              <Link href="/07-repo-detail" className="text-[#0969da] no-underline hover:underline"><strong>awesome-project</strong></Link>
              <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-[color:var(--info-light)] text-[color:var(--info)]">Public</span>
            </h1>
            <div className="flex gap-2 ml-auto">
              <Link href="/07-repo-detail" className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white hover:bg-gray-50 inline-flex items-center gap-1.5">👁 Watch <span className="bg-[#eaeef2] px-2 py-0.5 rounded-full text-xs">42</span></Link>
              <Link href="/07-repo-detail" className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white hover:bg-gray-50 inline-flex items-center gap-1.5">🍴 Fork <span className="bg-[#eaeef2] px-2 py-0.5 rounded-full text-xs">128</span></Link>
              <Link href="/07-repo-detail" className="px-3 py-1.5 text-sm bg-[color:var(--primary)] text-white rounded-md hover:bg-[color:var(--primary-hover)] inline-flex items-center gap-1.5">⭐ Star <span className="bg-white/20 px-2 py-0.5 rounded-full text-xs">2.4k</span></Link>
            </div>
          </div>

          <nav className="flex gap-1 mt-4">
            <Link href="/07-repo-detail" className="px-4 py-2 text-sm text-[#24292f] rounded-t-md inline-flex items-center gap-1.5 border-b-2 border-[#fd8c73] font-semibold">📄 Code</Link>
            <Link href="/12-issue-list" className="px-4 py-2 text-sm text-[#24292f] rounded-t-md hover:bg-gray-100 inline-flex items-center gap-1.5">⊙ Issues <span className="bg-[#eaeef2] px-2 py-0.5 rounded-full text-xs">23</span></Link>
            <Link href="/13-pr-list" className="px-4 py-2 text-sm text-[#24292f] rounded-t-md hover:bg-gray-100 inline-flex items-center gap-1.5">⇄ Pull requests <span className="bg-[#eaeef2] px-2 py-0.5 rounded-full text-xs">7</span></Link>
            <Link href="/10-actions-list" className="px-4 py-2 text-sm text-[#24292f] rounded-t-md hover:bg-gray-100 inline-flex items-center gap-1.5">▶ Actions</Link>
            <Link href="/09-commit-history" className="px-4 py-2 text-sm text-[#24292f] rounded-t-md hover:bg-gray-100 inline-flex items-center gap-1.5">⟳ Commits</Link>
            <Link href="/11-repo-settings" className="px-4 py-2 text-sm text-[#24292f] rounded-t-md hover:bg-gray-100 inline-flex items-center gap-1.5">⚙ Settings</Link>
          </nav>
        </div>
      </div>

      <div className="max-w-[1280px] mx-auto px-6">
        <div className="grid grid-cols-1 lg:grid-cols-[1fr_360px] gap-6 py-6">
          <div>
            <div className="bg-white border border-[#d0d7de] rounded-lg overflow-hidden">
              <div className="p-3 border-b border-[#d0d7de] flex items-center gap-3 flex-wrap">
                <Link href="/07-repo-detail" className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-[#f6f8fa] border border-[#d0d7de] rounded-md text-sm text-[#24292f] hover:bg-gray-100">⎇ <strong>main</strong> ▾</Link>
                <span className="text-[#57606a] text-[13px]">12 branches · 8 tags</span>
                <div className="ml-auto flex items-center gap-2">
                  <Link href="/09-commit-history" className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white hover:bg-gray-50">⟳ History</Link>
                  <CloneUrlToggle cloneUrl="https://example.com/octocat/awesome-project.git" sshUrl="git@example.com:octocat/awesome-project.git" />
                </div>
              </div>

              <ul className="list-none p-0 m-0">
                {files.map((f, i) => (
                  <li key={i}>
                    <Link href="/08-file-viewer" className="flex items-center px-4 py-2.5 border-b border-[#eaeef2] last:border-b-0 text-[#24292f] no-underline gap-3 text-sm hover:bg-[#f6f8fa]">
                      <span className="w-4 text-[#57606a]">{f.icon}</span>
                      <span className="flex-1">{f.name}</span>
                      <span className="text-[#57606a] text-[13px] flex-[2] overflow-hidden text-ellipsis whitespace-nowrap">{f.msg}</span>
                      <span className="text-[#57606a] text-[13px]">{f.age}</span>
                    </Link>
                  </li>
                ))}
              </ul>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-lg mt-6">
              <div className="p-3 border-b border-[#d0d7de] font-semibold flex items-center gap-2">📖 README.md</div>
              <div className="p-8">
                <h1 className="border-b border-[#d0d7de] pb-2 text-3xl">Awesome Project</h1>
                <p className="mt-3 flex gap-2 flex-wrap">
                  <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-[color:var(--success-light)] text-[color:var(--success)]">build passing</span>
                  <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-[color:var(--info-light)] text-[color:var(--info)]">coverage 94%</span>
                  <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-[color:var(--primary-light)] text-[color:var(--primary)]">v2.4.0</span>
                </p>
                <p className="mt-3 text-[color:var(--text-secondary)]">A modern, fast, and developer-friendly toolkit for building scalable web applications. This project provides a comprehensive set of utilities and components to streamline your development workflow.</p>

                <h2 className="mt-6 text-2xl">✨ Features</h2>
                <ul className="list-disc pl-6 mt-2 text-[color:var(--text-secondary)] space-y-1">
                  <li>🚀 Lightning-fast performance with minimal overhead</li>
                  <li>🔧 Modular architecture for easy customization</li>
                  <li>📦 Zero-config setup with sensible defaults</li>
                  <li>🧪 Comprehensive test coverage</li>
                  <li>📚 Detailed documentation and examples</li>
                </ul>

                <h2 className="mt-6 text-2xl">📥 Installation</h2>
                <pre className="bg-[#f6f8fa] p-4 rounded-md overflow-auto mt-2 font-mono text-sm"><code>{`npm install awesome-project
# or
yarn add awesome-project`}</code></pre>

                <h2 className="mt-6 text-2xl">🚀 Quick Start</h2>
                <pre className="bg-[#f6f8fa] p-4 rounded-md overflow-auto mt-2 font-mono text-sm"><code>{`import { createApp } from 'awesome-project';

const app = createApp({
  name: 'my-app',
  version: '1.0.0'
});

app.start();`}</code></pre>

                <h2 className="mt-6 text-2xl">📖 Documentation</h2>
                <p className="mt-2">For detailed documentation, please visit <Link href="/08-file-viewer" className="text-[#0969da] hover:underline">docs/</Link>.</p>

                <h2 className="mt-6 text-2xl">🤝 Contributing</h2>
                <p className="mt-2">Contributions are welcome! Please see <code className="bg-[#f6f8fa] px-1.5 py-0.5 rounded text-[90%]">CONTRIBUTING.md</code> for guidelines.</p>

                <h2 className="mt-6 text-2xl">📄 License</h2>
                <p className="mt-2">This project is licensed under the MIT License.</p>
              </div>
            </div>
          </div>

          <aside>
            <div className="bg-white border border-[#d0d7de] rounded-lg p-4 mb-4">
              <h3 className="text-sm m-0 mb-3 font-semibold">About</h3>
              <p className="text-[#57606a] text-sm leading-relaxed">A modern, fast, and developer-friendly toolkit for building scalable web applications.</p>
              <ul className="list-none p-0 mt-3 text-[13px] text-[#57606a] space-y-1">
                <li>🔗 <Link href="/07-repo-detail" className="text-[#0969da] no-underline hover:underline">awesome-project.dev</Link></li>
                <li>📜 MIT License</li>
                <li>⭐ 2,431 stars</li>
                <li>👁 42 watching</li>
                <li>🍴 128 forks</li>
              </ul>
              <div className="mt-3 flex flex-wrap gap-1.5">
                {["javascript", "typescript", "toolkit", "framework", "web"].map((t) => (
                  <span key={t} className="px-2 py-0.5 bg-[color:var(--info-light)] text-[color:var(--info)] rounded-full text-xs">{t}</span>
                ))}
              </div>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-lg p-4 mb-4">
              <h3 className="text-sm m-0 mb-3 font-semibold">Releases <span className="bg-[#eaeef2] px-2 py-0.5 rounded-full text-xs">12</span></h3>
              <div className="text-[13px]">
                <div className="mb-2"><strong>v2.4.0</strong> <span className="ml-1 px-2 py-0.5 rounded-full text-xs font-medium bg-[color:var(--success-light)] text-[color:var(--success)]">Latest</span></div>
                <div className="text-[color:var(--text-muted)]">Released 1 day ago</div>
              </div>
              <Link href="/07-repo-detail" className="text-[#0969da] no-underline text-[13px] block mt-3 hover:underline">+ 11 releases</Link>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-lg p-4 mb-4">
              <h3 className="text-sm m-0 mb-3 font-semibold">Packages</h3>
              <div className="text-[color:var(--text-muted)] text-[13px]">No packages published</div>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-lg p-4 mb-4">
              <h3 className="text-sm m-0 mb-3 font-semibold">Contributors <span className="bg-[#eaeef2] px-2 py-0.5 rounded-full text-xs">24</span></h3>
              <div className="flex gap-1 flex-wrap mt-2">
                {contributors.map((c, i) => (
                  <span key={i} className={`w-8 h-8 rounded-full ${c.bg} inline-flex items-center justify-center text-white text-xs font-semibold`}>{c.initials}</span>
                ))}
              </div>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-lg p-4 mb-4">
              <h3 className="text-sm m-0 mb-3 font-semibold">Languages</h3>
              <div className="h-2 rounded overflow-hidden flex my-2">
                {languages.map((l) => (
                  <div key={l.name} className={l.color} style={{ width: `${l.pct}%` }} />
                ))}
              </div>
              <div className="flex flex-wrap gap-3 text-xs">
                {languages.map((l) => (
                  <span key={l.name} className="inline-flex items-center">
                    <span className={`inline-block w-2.5 h-2.5 rounded-full mr-1 ${l.color}`}></span>
                    {l.name} {l.pct.toFixed(1)}%
                  </span>
                ))}
              </div>
            </div>
          </aside>
        </div>
      </div>
    </div>
  );
}
