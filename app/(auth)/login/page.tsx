import Link from "next/link";
import { LoginForm } from "@/components/auth/LoginForm";

export default function LoginPage() {
  return (
    <div className="min-h-screen bg-[#0d1117] text-[#c9d1d9] font-sans -m-6 w-[calc(100%+3rem)] max-w-none">
      <div className="max-w-[340px] mx-auto px-5 py-10">
        <div className="text-center mb-6">
          <span className="text-5xl text-[#c9d1d9]">⌥</span>
        </div>

        <h1 className="text-center text-2xl font-light mb-6 text-[#c9d1d9]">
          OpenHub にサインイン
        </h1>

        <div className="bg-[#161b22] border border-[#30363d] rounded-md p-4 mb-4">
          <LoginForm />
        </div>

        <div className="text-center p-4 bg-[#161b22] border border-[#30363d] rounded-md text-sm text-[#c9d1d9]">
          アカウントをお持ちでない方
          <Link
            href="/signup"
            className="text-[#58a6ff] no-underline ml-1 hover:underline"
          >
            新規登録
          </Link>
        </div>

        <div className="mt-12 text-center text-xs text-[#8b949e]">
          <Link href="/login" className="text-[#58a6ff] no-underline mx-2">
            利用規約
          </Link>
          <Link href="/login" className="text-[#58a6ff] no-underline mx-2">
            プライバシー
          </Link>
          <Link href="/login" className="text-[#58a6ff] no-underline mx-2">
            セキュリティ
          </Link>
          <Link href="/login" className="text-[#58a6ff] no-underline mx-2">
            ヘルプ
          </Link>
        </div>
      </div>
    </div>
  );
}
