"use client";

import Link from "next/link";
import { useState } from "react";

type FileItem = {
  icon: string;
  name: string;
  msg: string;
  age: string;
};

const files: FileItem[] = [
  { icon: "📁", name: ".github", msg: "Update workflow permissions", age: "3 days ago" },
  { icon: "📁", name: "src", msg: "Refactor authentication module", age: "5 hours ago" },
  { icon: "📁", name: "tests", msg: "Add integration tests for API", age: "2 days ago" },
  { icon: "📄", name: "README.md", msg: "Update installation guide", age: "3 days ago" },
  { icon: "📄", name: "package.json", msg: "Bump version to 2.4.0", age: "1 day ago" },
];

export default function RepoDetailPage() {
  const [cloneUrl] = useState("git@github.com:octocat/awesome-project.git");

  const handleCopy = () => {
    // TODO: wire to API
    navigator.clipboard?.writeText(cloneUrl);
  };

  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <header className="h-16 sticky top-0 z-[100] bg-white/85 backdrop-blur border-b border-[color:var(--border)]">
        <div className="max-w-[1280px] mx-auto px-6 flex items-center justify-between h-full">
          <div className="font-extrabold text-lg flex items-center gap-2">
            <span>🐙</span> OSS GitHub
          </div>
          <div className="flex items-center gap-3">
            <Link href="/12-issue-list" className="px-3 py-1.5 rounded-md hover:bg-[color:var(--bg-muted)] text-sm">
              🔔
            </Link>
            <Link href="/11-repo-settings" className="px-3 py-1.5 rounded-md hover:bg-[color:var(--bg-muted)] text-sm">
              ⚙️
            </Link>
            <span className="px-2.5 py-1 text-xs font-semibold rounded-full bg-[color:var(--primary-light)] text-[color:var(--primary)]">
              octocat
            </span>
          </div>
        </div>
      </header>

      <div className="bg-white border-b border-[#d0d7de] py-4 sticky top-16 z-10">
        <div className="max-w-[1280px] mx-auto px-6">
          <div className="flex items-center gap-3 flex-wrap">
            <h1 className="text-xl m-0 flex items-center gap-2 flex-wrap">
              <span>📁</span>
              <Link href="/07-repo-detail" className="text-[#0969da] hover:underline">
                octocat
              </Link>
              <span>/</span>
              <Link href="/07-repo-detail" className="text-[#0969da] hover:underline font-bold">
                awesome-project
              </Link>
              <span className="px-2 py-0.5 text-xs rounded-full bg-[color:var(--info-light)] text-[color:var(--info)]">
                Public
              </span>
            </h1>
            <div className="flex gap-2 ml-auto">
              <Link href="/07-repo-detail" className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-[#f6f8fa] hover:bg-gray-100 inline-flex items-center gap-1.5">
                👁 Watch <span className="bg-[#eaeef2] px-2 py-0.5 rounded-full text-xs">42</span>
              </Link>
              <Link href="/07-repo-detail" className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-[#f6f8fa] hover:bg-gray-100 inline-flex items-center gap-1.5">
                🍴 Fork <span className="bg-[#eaeef2] px-2 py-0.5 rounded-full text-xs">128</span>
              </Link>
              <Link href="/07-repo-detail" className="px-3 py-1.5 text-sm rounded-md bg-[color:var(--primary)] text-white hover:bg-[color:var(--primary-hover)] inline-flex items-center gap-1.5">
                ⭐ Star <span className="bg-white/20 px-2 py-0.5 rounded-full text-xs">2.4k</span>
              </Link>
            </div>
          </div>

          <nav className="flex gap-1 mt-4">
            <Link href="/07-repo-detail" className="px-4 py-2 text-sm rounded-t-md inline-flex items-center gap-1.5 border-b-2 border-[#fd8c73] font-semibold">
              📄 Code
            </Link>
            <Link href="/12-issue-list" className="px-4 py-2 text-sm rounded-t-md hover:bg-gray-100 inline-flex items-center gap-1.5">
              ⊙ Issues <span className="bg-[#eaeef2] px-2 py-0.5 rounded-full text-xs">23</span>
            </Link>
            <Link href="/13-pr-list" className="px-4 py-2 text-sm rounded-t-md hover:bg-gray-100 inline-flex items-center gap-1.5">
              ⇄ Pull requests <span className="bg-[#eaeef2] px-2 py-0.5 rounded-full text-xs">7</span>
            </Link>
            <Link href="/10-actions-list" className="px-4 py-2 text-sm rounded-t-md hover:bg-gray-100 inline-flex items-center gap-1.5">
              ▶ Actions
            </Link>
            <Link href="/09-commit-history" className="px-4 py-2 text-sm rounded-t-md hover:bg-gray-100 inline-flex items-center gap-1.5">
              ⟳ Commits
            </Link>
            <Link href="/11-repo-settings" className="px-4 py-2 text-sm rounded-t-md hover:bg-gray-100 inline-flex items-center gap-1.5">
              ⚙ Settings
            </Link>
          </nav>
        </div>
      </div>

      <div className="max-w-[1280px] mx-auto px-6">
        <div className="grid grid-cols-1 lg:grid-cols-[1fr_360px] gap-6 py-6">
          <div>
            <div className="bg-white border border-[#d0d7de] rounded-lg overflow-hidden">
              <div className="p-3 border-b border-[#d0d7de] flex items-center gap-3 flex-wrap">
                <Link href="/07-repo-detail" className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-[#f6f8fa] border border-[#d0d7de] rounded-md text-sm">
                  ⎇ <strong>main</strong> ▾
                </Link>
                <span className="text-[#57606a] text-[13px]">12 branches · 8 tags</span>
                <div className="ml-auto flex items-center gap-2">
                  <Link href="/09-commit-history" className="px-3 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-[#f6f8fa] hover:bg-gray-100">
                    ⟳ History
                  </Link>
                  <span className="bg-[#f6f8fa] border border-[#d0d7de] px-2.5 py-1.5 rounded-md font-mono text-xs min-w-[280px]">
                    {cloneUrl}
                  </span>
                  <button
                    onClick={handleCopy}
                    className="px-3 py-1.5 text-sm rounded-md bg-[color:var(--primary)] text-white hover:bg-[color:var(--primary-hover)]"
                  >
                    📋 Copy
                  </button>
                </div>
              </div>

              <ul>
                {files.map((f) => (
                  <li key={f.name}>
                    <Link
                      href="/08-file-viewer"
                      className="flex items-center px-4 py-2.5 border-b border-[#eaeef2] text-sm gap-3 hover:bg-[#f6f8fa] last:border-b-0"
                    >
                      <span className="w-4 text-[#57606a]">{f.icon}</span>
                      <span className="flex-1">{f.name}</span>
                      <span className="flex-[2] text-[#57606a] text-[13px] truncate">{f.msg}</span>
                      <span className="text-[#57606a] text-[13px]">{f.age}</span>
                    </Link>
                  </li>
                ))}
              </ul>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-lg mt-6">
              <div className="p-3 border-b border-[#d0d7de] font-semibold flex items-center gap-2">
                📖 README.md
              </div>
              <div className="p-8">
                <h1 className="text-2xl font-bold border-b border-[#d0d7de] pb-2 mb-4">Awesome Project</h1>
                <p className="flex gap-2 flex-wrap mb-4">
                  <span className="px-2 py-0.5 text-xs rounded-full bg-[color:var(--success-light)] text-[color:var(--success)]">
                    build passing
                  </span>
                  <span className="px-2 py-0.5 text-xs rounded-full bg-[color:var(--info-light)] text-[color:var(--info)]">
                    coverage 94%
                  </span>
                  <span className="px-2 py-0.5 text-xs rounded-full bg-[color:var(--primary-light)] text-[color:var(--primary)]">
                    v2.4.0
                  </span>
                </p>
                <p className="mb-4 text-[color:var(--text-secondary)]">
                  A modern, fast, and developer-friendly toolkit for building scalable web applications.
                </p>

                <h2 className="text-xl font-bold mt-6 mb-3">✨ Features</h2>
                <ul className="list-disc pl-6 mb-4 space-y-1 text-[color:var(--text-secondary)]">
                  <li>🚀 Lightning-fast performance with minimal overhead</li>
                  <li>🔧 Modular architecture for easy customization</li>
                  <li>📦 Zero-config setup with sensible defaults</li>
                  <li>🧪 Comprehensive test coverage</li>
                </ul>

                <h2 className="text-xl font-bold mt-6 mb-3">📥 Installation</h2>
                <pre className="bg-[#f6f8fa] p-4 rounded-md overflow-auto font-mono text-sm">
                  <code>{`npm install awesome-project
# or
yarn add awesome-project`}</code>
                </pre>

                <h2 className="text-xl font-bold mt-6 mb-3">🚀 Quick Start</h2>
                <pre className="bg-[#f6f8fa] p-4 rounded-md overflow-auto font-mono text-sm">
                  <code>{`import { createApp } from 'awesome-project';

const app = createApp({ name: 'my-app' });
app.start();`}</code>
                </pre>

                <h2 className="text-xl font-bold mt-6 mb-3">📄 License</h2>
                <p className="text-[color:var(--text-secondary)]">
                  This project is licensed under the MIT License.
                </p>
              </div>
            </div>
          </div>

          <aside>
            <div className="bg-white border border-[#d0d7de] rounded-lg p-4 mb-4">
              <h3 className="text-sm font-semibold mb-3">About</h3>
              <p className="text-[#57606a] text-sm leading-relaxed">
                A modern, fast, and developer-friendly toolkit for building scalable web applications.
              </p>
              <ul className="text-[13px] text-[#57606a] mt-3 space-y-1">
                <li>
                  🔗{" "}
                  <Link href="/07-repo-detail" className="text-[#0969da]">
                    awesome-project.dev
                  </Link>
                </li>
                <li>📜 MIT License</li>
                <li>⭐ 2,431 stars</li>
                <li>👁 42 watching</li>
                <li>🍴 128 forks</li>
              </ul>
              <div className="mt-3 flex flex-wrap gap-1.5">
                {["javascript", "typescript", "toolkit", "framework", "web"].map((t) => (
                  <span
                    key={t}
                    className="px-2 py-0.5 text-xs rounded-full bg-[color:var(--primary-light)] text-[color:var(--primary)]"
                  >
                    {t}
                  </span>
                ))}
              </div>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-lg p-4 mb-4">
              <h3 className="text-sm font-semibold mb-3">
                Releases{" "}
                <span className="bg-[#eaeef2] px-2 py-0.5 rounded-full text-xs">12</span>
              </h3>
              <div className="text-[13px]">
                <div className="mb-2">
                  <strong>v2.4.0</strong>{" "}
                  <span className="px-2 py-0.5 text-xs rounded-full bg-[color:var(--success-light)] text-[color:var(--success)]">
                    Latest
                  </span>
                </div>
                <div className="text-[color:var(--text-muted)]">Released 1 day ago</div>
              </div>
              <Link href="/07-repo-detail" className="text-[#0969da] text-[13px] block mt-3">
                + 11 releases
              </Link>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-lg p-4 mb-4">
              <h3 className="text-sm font-semibold mb-3">Packages</h3>
              <div className="text-[color:var(--text-muted)] text-[13px]">No packages published</div>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-lg p-4 mb-4">
              <h3 className="text-sm font-semibold mb-3">
                Contributors{" "}
                <span className="bg-[#eaeef2] px-2 py-0.5 rounded-full text-xs">24</span>
              </h3>
              <div className="flex gap-1 flex-wrap mt-2">
                <span className="w-8 h-8 rounded-full bg-[#f97583] inline-flex items-center justify-center text-white text-xs font-semibold">OC</span>
                <span className="w-8 h-8 rounded-full bg-[#79c0ff] inline-flex items-center justify-center text-white text-xs font-semibold">JD</span>
                <span className="w-8 h-8 rounded-full bg-[#a5d6ff] inline-flex items-center justify-center text-white text-xs font-semibold">AM</span>
                <span className="w-8 h-8 rounded-full bg-[#d2a8ff] inline-flex items-center justify-center text-white text-xs font-semibold">KS</span>
                <span className="w-8 h-8 rounded-full bg-[#ffa657] inline-flex items-center justify-center text-white text-xs font-semibold">RT</span>
                <span className="w-8 h-8 rounded-full bg-[#7ee787] inline-flex items-center justify-center text-white text-xs font-semibold">+19</span>
              </div>
            </div>

            <div className="bg-white border border-[#d0d7de] rounded-lg p-4 mb-4">
              <h3 className="text-sm font-semibold mb-3">Languages</h3>
              <div className="h-2 rounded-full overflow-hidden flex my-2">
                <div className="w-[65%] bg-[#3178c6]" />
                <div className="w-[22%] bg-[#f1e05a]" />
                <div className="w-[8%] bg-[#563d7c]" />
                <div className="w-[5%] bg-[#89e051]" />
              </div>
              <div className="flex flex-wrap gap-3 text-xs">
                <span className="flex items-center gap-1">
                  <span className="w-2.5 h-2.5 rounded-full bg-[#3178c6] inline-block" />
                  TypeScript 65.0%
                </span>
                <span className="flex items-center gap-1">
                  <span className="w-2.5 h-2.5 rounded-full bg-[#f1e05a] inline-block" />
                  JavaScript 22.0%
                </span>
                <span className="flex items-center gap-1">
                  <span className="w-2.5 h-2.5 rounded-full bg-[#563d7c] inline-block" />
                  CSS 8.0%
                </span>
                <span className="flex items-center gap-1">
                  <span className="w-2.5 h-2.5 rounded-full bg-[#89e051] inline-block" />
                  Shell 5.0%
                </span>
              </div>
            </div>
          </aside>
        </div>
      </div>
    </div>
  );
}
