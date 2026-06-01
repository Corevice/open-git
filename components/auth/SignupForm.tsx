"use client";

import { useState, type FormEvent } from "react";
import { apiClient } from "@/lib/api-client";
import { ApiError } from "@/lib/api";

const USERNAME_PATTERN = /^[a-zA-Z0-9-]{3,39}$/;

type FieldErrors = {
  username?: string;
  email?: string;
  password?: string;
  general?: string;
};

type ApiFieldError = {
  field?: string;
  resource?: string;
  message: string;
};

type RegisterErrorBody = {
  message?: string;
  errors?: ApiFieldError[];
};

function mapErrorsToFields(errors: ApiFieldError[]): FieldErrors {
  const result: FieldErrors = {};
  for (const err of errors) {
    const field = (err.field ?? "").toLowerCase();
    if (field === "login" || field === "username") {
      result.username = err.message;
    } else if (field === "email") {
      result.email = err.message;
    } else if (field === "password") {
      result.password = err.message;
    } else {
      result.general = err.message;
    }
  }
  return result;
}

async function fetchRegisterErrorBody(
  body: { login: string; email: string; password: string },
): Promise<RegisterErrorBody | null> {
  const baseURL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
  try {
    const response = await fetch(`${baseURL}/api/v1/auth/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    return (await response.json()) as RegisterErrorBody;
  } catch {
    return null;
  }
}

export function SignupForm() {
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [successMessage, setSuccessMessage] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const inputClass =
    "w-full px-3 py-2 border border-[color:var(--border)] rounded-md text-sm bg-white focus:outline-none focus:border-[color:var(--primary)] focus:shadow-[0_0_0_3px_rgba(99,102,241,0.2)]";
  const labelClass =
    "block text-sm font-medium mb-1.5 text-[color:var(--text-primary)]";
  const hintClass = "text-xs text-[color:var(--text-muted)] mt-1";
  const errorClass = "text-xs text-[color:var(--danger)] mt-1";

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setFieldErrors({});
    setSuccessMessage("");

    const nextErrors: FieldErrors = {};

    if (!USERNAME_PATTERN.test(username)) {
      nextErrors.username =
        "ユーザー名は英数字とハイフンのみ、3〜39文字で入力してください。";
    }

    if (!email.trim()) {
      nextErrors.email = "メールアドレスを入力してください。";
    }

    if (password.length < 8) {
      nextErrors.password = "パスワードは8文字以上で入力してください。";
    }

    if (Object.keys(nextErrors).length > 0) {
      setFieldErrors(nextErrors);
      return;
    }

    const payload = {
      login: username,
      email: email.trim(),
      password,
    };

    setSubmitting(true);
    try {
      await apiClient.post("/api/v1/auth/register", payload);
      setSuccessMessage("アカウントを作成しました。サインインしてください。");
    } catch (err) {
      if (err instanceof ApiError && err.status === 422) {
        const body = await fetchRegisterErrorBody(payload);
        if (body?.errors && body.errors.length > 0) {
          setFieldErrors(mapErrorsToFields(body.errors));
          return;
        }
        setFieldErrors({ general: err.message });
        return;
      }

      if (err instanceof ApiError) {
        setFieldErrors({ general: err.message });
      } else {
        setFieldErrors({ general: "登録に失敗しました。" });
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <div className="mb-4">
        <label className={labelClass} htmlFor="username">
          ユーザー名 <span className="text-[color:var(--danger)]">*</span>
        </label>
        <input
          type="text"
          id="username"
          className={inputClass}
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          placeholder="例: octocat-dev"
        />
        <div className={hintClass}>英数字とハイフンが使用できます。3〜39文字。</div>
        {fieldErrors.username ? (
          <p className={errorClass} role="alert">
            {fieldErrors.username}
          </p>
        ) : null}
      </div>

      <div className="mb-4">
        <label className={labelClass} htmlFor="email">
          メールアドレス <span className="text-[color:var(--danger)]">*</span>
        </label>
        <input
          type="email"
          id="email"
          className={inputClass}
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="you@example.com"
        />
        {fieldErrors.email ? (
          <p className={errorClass} role="alert">
            {fieldErrors.email}
          </p>
        ) : null}
      </div>

      <div className="mb-4">
        <label className={labelClass} htmlFor="password">
          パスワード <span className="text-[color:var(--danger)]">*</span>
        </label>
        <input
          type="password"
          id="password"
          className={inputClass}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        <div className={hintClass}>8文字以上で入力してください。</div>
        {fieldErrors.password ? (
          <p className={errorClass} role="alert">
            {fieldErrors.password}
          </p>
        ) : null}
      </div>

      {fieldErrors.general ? (
        <p className={`${errorClass} mb-4`} role="alert">
          {fieldErrors.general}
        </p>
      ) : null}

      {successMessage ? (
        <p className="mb-4 text-sm text-[color:var(--success)]" role="status">
          {successMessage}
        </p>
      ) : null}

      <button
        type="submit"
        disabled={submitting}
        className="w-full inline-flex items-center justify-center gap-1.5 px-4 py-2.5 rounded-md text-base font-medium transition bg-[color:var(--primary)] text-white hover:bg-[color:var(--primary-hover)] disabled:opacity-60"
      >
        {submitting ? "作成中..." : "アカウントを作成 →"}
      </button>
    </form>
  );
}
