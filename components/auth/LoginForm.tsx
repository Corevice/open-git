"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { apiClient } from "@/lib/api-client";
import { useAuth } from "@/lib/auth";
import { ApiError } from "@/lib/api";

type LoginResponse = {
  token: string;
};

export function LoginForm() {
  const [login, setLogin] = useState("");
  const [password, setPassword] = useState("");
  const [errorMessage, setErrorMessage] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const router = useRouter();
  const auth = useAuth();

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setErrorMessage("");

    if (!login.trim() || !password) {
      setErrorMessage("ユーザー名とパスワードを入力してください。");
      return;
    }

    setSubmitting(true);
    try {
      const response = await apiClient.post<LoginResponse>(
        "/api/v1/auth/login",
        { login: login.trim(), password },
      );
      auth.login(response.token);
      document.cookie = `authToken=${encodeURIComponent(response.token)}; path=/; SameSite=Lax`;
      router.push("/dashboard");
    } catch (err) {
      if (err instanceof ApiError) {
        setErrorMessage(err.message || "サインインに失敗しました。");
      } else {
        setErrorMessage("サインインに失敗しました。");
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <div className="mb-4">
        <label
          className="block text-sm font-semibold mb-1.5 text-[#c9d1d9]"
          htmlFor="login"
        >
          ユーザー名またはメールアドレス
        </label>
        <input
          type="text"
          id="login"
          name="login"
          value={login}
          onChange={(e) => setLogin(e.target.value)}
          className="w-full box-border px-3 py-2 bg-[#0d1117] border border-[#30363d] rounded-md text-[#c9d1d9] text-sm leading-5"
          autoComplete="username"
        />
      </div>

      <div className="mb-4">
        <label
          className="block text-sm font-semibold mb-1.5 text-[#c9d1d9]"
          htmlFor="password"
        >
          パスワード
        </label>
        <input
          type="password"
          id="password"
          name="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          className="w-full box-border px-3 py-2 bg-[#0d1117] border border-[#30363d] rounded-md text-[#c9d1d9] text-sm leading-5"
          autoComplete="current-password"
        />
      </div>

      <button
        type="submit"
        disabled={submitting}
        className="w-full bg-[#238636] hover:bg-[#2ea043] disabled:opacity-60 text-white border border-white/10 px-4 py-2 rounded-md text-sm font-semibold cursor-pointer text-center block box-border"
      >
        {submitting ? "サインイン中..." : "サインイン"}
      </button>

      {errorMessage ? (
        <p className="mt-3 text-sm text-[#f85149]" role="alert">
          {errorMessage}
        </p>
      ) : null}
    </form>
  );
}
