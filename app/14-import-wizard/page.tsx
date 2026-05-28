"use client";

import Link from "next/link";
import { useState } from "react";

type ImportItem = {
  id: string;
  icon: string;
  title: string;
  desc: string;
  count: string;
};

const items: ImportItem[] = [
  { id: "code", icon: "📂", title: "Code（ソースコード）", desc: "全ブランチ・タグ・コミット履歴を含む完全な複製", count: "推定サイズ: 245 MB / 38,420 コミット" },
  { id: "issues", icon: "🐛", title: "Issues", desc: "課題・ラベル・コメント・添付ファイル", count: "12,847 件のIssue（うちOpen: 1,283）" },
  { id: "prs", icon: "🔀", title: "Pull Requests", desc: "PR・レビュー・コメント・マージ履歴", count: "8,921 件のPR（うちOpen: 412）" },
  { id: "wiki", icon: "📖", title: "Wiki", desc: "Wikiページとリビジョン履歴", count: "86 ページ" },
  { id: "releases", icon: "🚀", title: "Releases", desc: "リリースタグとリリースノート", count: "142 リリース" },
];

export default function Page() {
  const [checked, setChecked] = useState<Record<string, boolean>>({
    code: true,
    issues: true,
    prs: true,
    wiki: false,
    releases: false,
  });

  const toggle = (id: string) => {
    setChecked((s) => ({ ...s, [id]: !s[id] }));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: wire to API
  };

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      {/* App bar */}
      <div className="bg-[#24292f] text-white px-6 py-3 flex items-center justify-between">
        <div className="flex items-center gap-2 font-semibold">
          <span className="text-xl">🐙</span>
          <span>OpenSource GitHub</span>
        </div>
        <div>
          <Link href="/05-repo-list" className="text-[#c9d1d9] px-4 py-2 text-sm">
            マイリポジトリ
          </Link>
        </div>
      </div>

      <div className="max-w-[1280px] mx-auto px-6 py-8">
        {/* Header */}
        <div className="text-center mb-10">
          <h1 className="text-[28px] font-bold mb-2">📥 GitHubリポジトリのインポート</h1>
          <p className="text-[#656d76]">既存のGitHubリポジトリをこのプラットフォームに移行します</p>
        </div>

        {/* Stepper */}
        <div className="flex justify-center items-center max-w-[720px] mx-auto mb-10">
          <div className="flex flex-col items-center min-w-[120px]">
            <div className="w-10 h-10 rounded-full flex items-center justify-center font-semibold bg-[#1f883d] text-white border-2 border-[#1f883d]">✓</div>
            <div className="mt-2 text-[13px] font-medium text-[#1f883d]">URL入力</div>
          </div>
          <div className="flex-1 h-[2px] bg-[#1f883d] -mx-5 -mt-7 self-center min-w-[40px]" />
          <div className="flex flex-col items-center min-w-[120px]">
            <div className="w-10 h-10 rounded-full flex items-center justify-center font-semibold bg-[#1f883d] text-white border-2 border-[#1f883d]">✓</div>
            <div className="mt-2 text-[13px] font-medium text-[#1f883d]">認証</div>
          </div>
          <div className="flex-1 h-[2px] bg-[#1f883d] -mx-5 -mt-7 self-center min-w-[40px]" />
          <div className="flex flex-col items-center min-w-[120px]">
            <div className="w-10 h-10 rounded-full flex items-center justify-center font-semibold bg-[#0969da] text-white border-2 border-[#0969da] shadow-[0_0_0_4px_rgba(9,105,218,0.15)]">3</div>
            <div className="mt-2 text-[13px] font-semibold text-[#0969da]">移行範囲</div>
          </div>
          <div className="flex-1 h-[2px] bg-[#eaeef2] -mx-5 -mt-7 self-center min-w-[40px]" />
          <div className="flex flex-col items-center min-w-[120px]">
            <div className="w-10 h-10 rounded-full flex items-center justify-center font-semibold bg-[#eaeef2] text-[#656d76] border-2 border-[#eaeef2]">4</div>
            <div className="mt-2 text-[13px] font-medium text-[#656d76]">インポート</div>
          </div>
        </div>

        {/* Wizard card */}
        <form onSubmit={handleSubmit} className="max-w-[600px] mx-auto bg-white border border-[#d0d7de] rounded-lg overflow-hidden">
          <div className="px-6 py-5 border-b border-[#d0d7de]">
            <h2 className="text-[18px] font-semibold mb-1">移行対象を選択</h2>
            <p className="text-[#656d76] text-sm">インポートするデータの種類を選んでください</p>
          </div>
          <div className="p-6">
            <div className="bg-[#ddf4ff] border border-[#54aeff66] rounded-md px-4 py-3 text-[13px] text-[#0550ae] mb-5 flex gap-2">
              <span>ℹ️</span>
              <div>
                リポジトリ <strong>facebook/react</strong> から以下のデータを検出しました。必要な項目のみインポートできます。
              </div>
            </div>

            <div className="flex flex-col gap-3">
              {items.map((it) => (
                <label key={it.id} className="flex items-start gap-3 px-4 py-3 border border-[#d0d7de] rounded-md cursor-pointer hover:bg-[#f6f8fa]">
                  <input
                    type="checkbox"
                    checked={!!checked[it.id]}
                    onChange={() => toggle(it.id)}
                    className="mt-1"
                  />
                  <div className="flex-1">
                    <div className="font-semibold text-sm">
                      {it.icon} {it.title}
                    </div>
                    <div className="text-xs text-[#656d76] mt-0.5">{it.desc}</div>
                    <div className="text-xs text-[#0969da] font-medium mt-1">{it.count}</div>
                  </div>
                </label>
              ))}
            </div>

            <div className="mt-5 p-3 bg-[#f6f8fa] rounded-md text-[13px]">
              <strong>推定インポート時間:</strong> 約 15〜20 分
              <br />
              <span className="text-[#656d76]">バックグラウンドで実行され、完了時に通知されます</span>
            </div>
          </div>
        </form>

        {/* Footer */}
        <div className="max-w-[600px] mx-auto mt-4 flex justify-between items-center">
          <Link href="/05-repo-list" className="inline-flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium text-[#656d76]">
            ✕ キャンセル
          </Link>
          <div className="flex gap-2">
            <Link
              href="/14-import-wizard"
              className="inline-flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium bg-white text-[#1f2328] border border-[#d0d7de]"
            >
              ← 戻る
            </Link>
            <Link
              href="/07-repo-detail"
              className="inline-flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium bg-[#1f883d] text-white border border-[rgba(31,35,40,0.15)]"
            >
              インポート実行 →
            </Link>
          </div>
        </div>

        {/* Preview details */}
        <div className="max-w-[600px] mx-auto mt-10">
          <details className="bg-white border border-[#d0d7de] rounded-lg px-4 py-3">
            <summary className="cursor-pointer font-semibold text-[#656d76] text-[13px]">
              👁 ステップ4プレビュー（実行中の表示）
            </summary>
            <div className="mt-4 text-center">
              <h3 className="text-base font-bold mb-1">インポート実行中...</h3>
              <p className="text-[13px] text-[#656d76]">facebook/react をインポートしています</p>
              <div className="w-full h-2 bg-[#eaeef2] rounded my-5 overflow-hidden">
                <div className="h-full w-[65%] bg-gradient-to-r from-[#0969da] to-[#218bff] transition-all" />
              </div>
              <div className="text-[13px] text-[#656d76] mb-6">
                <strong>65%</strong> 完了 — 残り 約 6 分
              </div>
              <div className="bg-[#0d1117] text-[#c9d1d9] p-4 rounded-md font-mono text-xs text-left max-h-[200px] overflow-y-auto leading-7">
                <div className="text-[#56d364]">[✓] リポジトリ初期化完了</div>
                <div className="text-[#56d364]">[✓] Code: 38,420 コミットをクローン</div>
                <div className="text-[#56d364]">[✓] Issues: 12,847 件をインポート</div>
                <div className="text-[#58a6ff]">[→] Pull Requests を処理中... (5,234 / 8,921)</div>
                <div className="text-[#d29922]">[…] レビューコメントを変換しています</div>
              </div>
            </div>
          </details>
        </div>
      </div>
    </div>
  );
}
