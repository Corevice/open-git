'use client';

import { Suspense, useEffect, useMemo, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { ApiClient, ApiError } from '@/lib/api';
import type { OAuthApp } from '@/lib/api-types';
import { useAuth } from '@/lib/auth';

const SCOPE_LABELS: Record<string, string> = {
  repo: 'リポジトリへの読み書きアクセス',
  'read:user': 'プロフィール情報の読み取り',
  'user:email': 'メールアドレスの読み取り',
  'admin:org': 'Organization の管理',
  workflow: 'GitHub Actions Workflow の管理',
};

function getHostname(uri: string): string {
  try {
    return new URL(uri).hostname;
  } catch {
    return uri;
  }
}

function apiBaseURL(): string {
  return (
    process.env.NEXT_PUBLIC_API_BASE_URL ??
    process.env.NEXT_PUBLIC_API_URL ??
    'http://localhost:8080'
  );
}

function ErrorCard({ message }: { message: string }) {
  return (
    <div className="bg-[#161b22] border border-[#f85149] rounded-md p-4">
      <p className="text-[#f85149] text-sm" role="alert">
        {message}
      </p>
    </div>
  );
}

function OAuthAuthorizePageContent() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const { token } = useAuth();

  const clientId = searchParams.get('client_id') ?? '';
  const redirectUri = searchParams.get('redirect_uri') ?? '';
  const scope = searchParams.get('scope') ?? '';
  const state = searchParams.get('state') ?? '';
  const responseType = searchParams.get('response_type') ?? '';

  const returnTo = useMemo(() => {
    const params = new URLSearchParams();
    if (clientId) params.set('client_id', clientId);
    if (redirectUri) params.set('redirect_uri', redirectUri);
    if (scope) params.set('scope', scope);
    if (state) params.set('state', state);
    if (responseType) params.set('response_type', responseType);
    return `/login/oauth/authorize?${params.toString()}`;
  }, [clientId, redirectUri, scope, state, responseType]);

  const [app, setApp] = useState<OAuthApp | null>(null);
  const [loading, setLoading] = useState(true);
  const [fetchError, setFetchError] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  // A plain form POST cannot carry the Bearer token, so approval calls the
  // backend authorize endpoint via fetch (Accept: application/json makes it
  // return the redirect target instead of a 302) and then navigates there.
  const handleApprove = async () => {
    setSubmitting(true);
    setSubmitError(null);
    try {
      const params = new URLSearchParams({
        client_id: clientId,
        redirect_uri: redirectUri,
        scope,
        state,
      });
      const res = await fetch(`${apiBaseURL()}/login/oauth/authorize?${params.toString()}`, {
        headers: {
          Authorization: `Bearer ${token}`,
          Accept: 'application/json',
        },
      });
      if (!res.ok) {
        setSubmitError('認可に失敗しました。時間をおいて再度お試しください。');
        return;
      }
      const body = (await res.json()) as { redirect_url?: string };
      if (!body.redirect_url) {
        setSubmitError('認可に失敗しました。時間をおいて再度お試しください。');
        return;
      }
      window.location.assign(body.redirect_url);
    } catch {
      setSubmitError('認可に失敗しました。時間をおいて再度お試しください。');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeny = () => {
    try {
      const target = new URL(redirectUri);
      target.searchParams.set('error', 'access_denied');
      if (state) target.searchParams.set('state', state);
      window.location.assign(target.toString());
    } catch {
      router.push('/dashboard');
    }
  };

  useEffect(() => {
    if (!token) {
      router.replace(`/login?return_to=${encodeURIComponent(returnTo)}`);
    }
  }, [token, router, returnTo]);

  useEffect(() => {
    if (!token || !clientId || responseType !== 'code') {
      setLoading(false);
      return;
    }

    const baseURL =
      process.env.NEXT_PUBLIC_API_BASE_URL ??
      process.env.NEXT_PUBLIC_API_URL ??
      'http://localhost:8080';
    const client = new ApiClient(baseURL);
    client.setToken(token);

    let cancelled = false;
    setLoading(true);
    setFetchError(false);

    client.oauthApps
      .get(clientId)
      .then((oauthApp) => {
        if (!cancelled) {
          setApp(oauthApp);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setFetchError(err instanceof ApiError && err.status === 404);
          setApp(null);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [token, clientId, responseType]);

  if (!token) {
    return null;
  }

  if (responseType !== 'code') {
    return (
      <div className="min-h-screen bg-[#0d1117] text-[#c9d1d9] font-sans -m-6 w-[calc(100%+3rem)] max-w-none">
        <div className="max-w-[480px] mx-auto px-5 py-10">
          <ErrorCard message="unsupported_response_type" />
        </div>
      </div>
    );
  }

  if (!clientId || fetchError || (!loading && !app)) {
    return (
      <div className="min-h-screen bg-[#0d1117] text-[#c9d1d9] font-sans -m-6 w-[calc(100%+3rem)] max-w-none">
        <div className="max-w-[480px] mx-auto px-5 py-10">
          <ErrorCard message="不正なリクエスト: client_id が無効です" />
        </div>
      </div>
    );
  }

  if (loading || !app) {
    return (
      <div className="min-h-screen bg-[#0d1117] text-[#c9d1d9] font-sans -m-6 w-[calc(100%+3rem)] max-w-none">
        <div className="max-w-[480px] mx-auto px-5 py-10">
          <p className="text-sm text-[#8b949e]">読み込み中...</p>
        </div>
      </div>
    );
  }

  if (!redirectUri || !app.callback_urls.includes(redirectUri)) {
    return (
      <div className="min-h-screen bg-[#0d1117] text-[#c9d1d9] font-sans -m-6 w-[calc(100%+3rem)] max-w-none">
        <div className="max-w-[480px] mx-auto px-5 py-10">
          <ErrorCard message="不正な redirect_uri" />
        </div>
      </div>
    );
  }

  const scopes = scope.split(/[\s,]+/).filter(Boolean);

  return (
    <div className="min-h-screen bg-[#0d1117] text-[#c9d1d9] font-sans -m-6 w-[calc(100%+3rem)] max-w-none">
      <div className="max-w-[480px] mx-auto px-5 py-10">
        <div className="bg-[#161b22] border border-[#30363d] rounded-md p-4">
          <h1 className="text-xl font-semibold mb-2 text-[#c9d1d9]">{app.name}</h1>

          {app.homepage_url ? (
            <p className="mb-4 text-sm">
              <a
                href={app.homepage_url}
                className="text-[#58a6ff] no-underline hover:underline"
                target="_blank"
                rel="noopener noreferrer"
              >
                {app.homepage_url}
              </a>
            </p>
          ) : null}

          <p className="text-sm text-[#8b949e] mb-4">このアプリは以下の権限を要求しています:</p>

          <ul className="mb-4 space-y-2 text-sm">
            {scopes.map((s) => (
              <li key={s} className="flex items-start gap-2">
                <span className="text-[#58a6ff]">•</span>
                <span>{SCOPE_LABELS[s] ?? s}</span>
              </li>
            ))}
          </ul>

          <p className="text-sm text-[#8b949e] mb-6">
            認可後、{getHostname(redirectUri)} へリダイレクトされます
          </p>

          {submitError ? (
            <p className="text-[#f85149] text-sm mb-4" role="alert">
              {submitError}
            </p>
          ) : null}

          <div className="flex gap-3">
            <button
              type="button"
              onClick={handleApprove}
              disabled={submitting}
              className="flex-1 bg-[#238636] hover:bg-[#2ea043] disabled:opacity-60 text-white border border-white/10 px-4 py-2 rounded-md text-sm font-semibold cursor-pointer"
            >
              {submitting ? '処理中...' : '許可'}
            </button>

            <button
              type="button"
              onClick={handleDeny}
              disabled={submitting}
              className="flex-1 bg-[#21262d] hover:bg-[#30363d] disabled:opacity-60 text-[#c9d1d9] border border-[#30363d] px-4 py-2 rounded-md text-sm font-semibold cursor-pointer"
            >
              拒否
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function OAuthAuthorizePage() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen bg-[#0d1117] text-[#c9d1d9] font-sans -m-6 w-[calc(100%+3rem)] max-w-none">
          <div className="max-w-[480px] mx-auto px-5 py-10">
            <p className="text-sm text-[#8b949e]">読み込み中...</p>
          </div>
        </div>
      }
    >
      <OAuthAuthorizePageContent />
    </Suspense>
  );
}
