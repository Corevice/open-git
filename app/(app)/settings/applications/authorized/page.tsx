"use client";

import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ApiClient } from "@/lib/api";
import type { OAuthAuthorizationInfo } from "@/lib/api-types";
import { useAuth } from "@/lib/auth";

const SCOPE_LABELS: Record<string, string> = {
  repo: "Full control of private repositories",
  "read:user": "Read user profile data",
  "user:email": "Access user email addresses",
  "admin:org": "Full control of organizations",
  workflow: "Update GitHub Action workflows",
  "read:org": "Read org and team membership",
  "write:packages": "Upload packages",
  "read:packages": "Download packages",
  gist: "Create gists",
  notifications: "Access notifications",
};

function formatScope(scope: string): string {
  return SCOPE_LABELS[scope] ?? scope;
}

function formatDate(value: string | null): string {
  if (!value) {
    return "—";
  }
  return new Date(value).toLocaleString();
}

type AuthorizationRow = OAuthAuthorizationInfo & {
  homepage_url?: string;
};

export default function AuthorizedApplicationsPage() {
  const { token, isAuthenticated } = useAuth();
  const router = useRouter();
  const baseURL =
    process.env.NEXT_PUBLIC_API_BASE_URL ??
    process.env.NEXT_PUBLIC_API_URL ??
    "http://localhost:8080";

  const apiClient = useMemo(() => new ApiClient(baseURL), [baseURL]);

  const [authorizations, setAuthorizations] = useState<AuthorizationRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [revokingId, setRevokingId] = useState<string | null>(null);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push("/login");
    }
  }, [isAuthenticated, router]);

  useEffect(() => {
    apiClient.setToken(token);
  }, [apiClient, token]);

  useEffect(() => {
    if (!isAuthenticated || !token) {
      return;
    }

    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(null);
      try {
        const list = await apiClient.userAuthorizations.list();
        if (!cancelled) {
          setAuthorizations(list);
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error
              ? err.message
              : "認可済みアプリケーションの読み込みに失敗しました。",
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [apiClient, isAuthenticated, token]);

  const handleRevoke = async (oauthAppId: string) => {
    setRevokingId(oauthAppId);
    setError(null);

    try {
      await apiClient.userAuthorizations.revoke(oauthAppId);
      setAuthorizations((prev) =>
        prev.filter((item) => item.oauth_app_id !== oauthAppId),
      );
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "連携解除に失敗しました。",
      );
    } finally {
      setRevokingId(null);
    }
  };

  if (!isAuthenticated) {
    return null;
  }

  return (
    <main className="mx-auto max-w-[960px] px-6 py-8">
      <div className="mb-6 border-b border-[#d1d9e0] pb-4">
        <h1 className="mb-2 text-2xl font-semibold">認可済みアプリケーション</h1>
        <p className="text-sm text-[#59636e]">
          あなたのアカウントへのアクセスを許可した OAuth App
          です。連携解除すると、当該アプリが発行したアクセストークンは直ちに失効します。
        </p>
      </div>

      {error && (
        <p className="mb-4 text-sm text-[#cf222e]" role="alert">
          {error}
        </p>
      )}

      {loading ? (
        <p className="text-sm text-[#59636e]">読み込み中…</p>
      ) : authorizations.length === 0 ? (
        <p className="text-sm text-[#59636e]">
          認可済みの OAuth App はありません
        </p>
      ) : (
        <div className="overflow-x-auto rounded-lg border border-[#d1d9e0]">
          <table className="min-w-full text-left text-sm">
            <thead className="border-b border-[#d1d9e0] bg-[#f6f8fa]">
              <tr>
                <th className="px-4 py-3 font-semibold">アプリ名</th>
                <th className="px-4 py-3 font-semibold">付与済みスコープ</th>
                <th className="px-4 py-3 font-semibold">最終更新</th>
                <th className="px-4 py-3 font-semibold">操作</th>
              </tr>
            </thead>
            <tbody>
              {authorizations.map((item) => (
                <tr
                  key={item.oauth_app_id}
                  className="border-b border-[#d1d9e0] last:border-b-0"
                >
                  <td className="px-4 py-3 font-medium">
                    {item.homepage_url ? (
                      <a
                        href={item.homepage_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-[#0969da] hover:underline"
                      >
                        {item.app_name}
                      </a>
                    ) : (
                      item.app_name
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      {item.granted_scopes.map((scope) => (
                        <Badge key={scope} variant="secondary">
                          {formatScope(scope)}
                        </Badge>
                      ))}
                    </div>
                  </td>
                  <td className="px-4 py-3">{formatDate(item.updated_at)}</td>
                  <td className="px-4 py-3">
                    <Button
                      type="button"
                      variant="destructive"
                      size="sm"
                      onClick={() => handleRevoke(item.oauth_app_id)}
                      disabled={revokingId === item.oauth_app_id}
                    >
                      {revokingId === item.oauth_app_id
                        ? "解除中…"
                        : "連携解除"}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </main>
  );
}
