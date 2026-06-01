import Link from "next/link";
import { SignupForm } from "@/components/auth/SignupForm";

export default function SignupPage() {
  return (
    <div className="min-h-screen bg-[color:var(--bg-base)] -m-6 w-[calc(100%+3rem)] max-w-none">
      <header className="h-16 bg-white/85 backdrop-blur border-b border-[color:var(--border)] flex items-center justify-between px-6 sticky top-0 z-10">
        <Link
          href="/"
          className="flex items-center gap-2 font-extrabold text-lg text-[color:var(--text-primary)] no-underline"
        >
          <span className="text-xl">🐙</span>
          <span>OctoHub</span>
        </Link>
        <div className="flex items-center gap-4">
          <Link
            href="/login"
            className="px-3 py-1.5 text-sm text-[color:var(--text-secondary)] hover:text-[color:var(--primary)]"
          >
            サインイン
          </Link>
        </div>
      </header>

      <div className="max-w-[640px] mx-auto px-6 pt-8 pb-16">
        <div className="text-center mb-8">
          <div className="text-5xl leading-none">🐙</div>
          <div className="text-2xl font-semibold mt-3 mb-1">OctoHubへようこそ</div>
          <div className="text-[color:var(--text-secondary)] text-sm">
            数百万の開発者と一緒に、コードで世界を変えよう
          </div>
        </div>

        <div className="bg-white border border-[color:var(--border)] rounded-lg shadow-sm overflow-hidden">
          <div className="px-6 py-4 border-b border-[color:var(--border)]">
            <div className="text-base font-semibold">アカウントを作成</div>
          </div>
          <div className="px-6 py-5">
            <SignupForm />
          </div>
          <div className="px-6 py-4 border-t border-[color:var(--border)] bg-[color:var(--bg-muted)]">
            <div className="text-center text-sm text-[color:var(--text-secondary)]">
              すでにアカウントをお持ちですか？{" "}
              <Link
                href="/login"
                className="text-[color:var(--primary)] font-medium no-underline"
              >
                サインイン
              </Link>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
