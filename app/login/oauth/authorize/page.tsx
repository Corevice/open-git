"use client";

import { useSearchParams } from "next/navigation";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { env } from "@/lib/env";

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

function ErrorCard({ message }: { message: string }) {
  return (
    <div className="min-h-screen bg-[#0d1117] text-[#c9d1d9] font-sans flex items-center justify-center p-6">
      <div className="w-full max-w-[480px]">
        <div className="bg-[#161b22] border border-[#f85149] rounded-md p-4">
          <p className="text-[#f85149] text-sm" role="alert">
            {message}
          </p>
        </div>
      </div>
    </div>
  );
}

export default function OAuthAuthorizePage() {
  const searchParams = useSearchParams();

  const clientId = searchParams.get("client_id") ?? "";
  const scope = searchParams.get("scope") ?? "";
  const state = searchParams.get("state") ?? "";
  const redirectUri = searchParams.get("redirect_uri") ?? "";

  if (!clientId || !state) {
    return (
      <ErrorCard message="Invalid request: client_id and state are required." />
    );
  }

  const scopes = scope.split(/[\s,]+/).filter(Boolean);

  const handleAuthorize = () => {
    const params = new URLSearchParams({
      client_id: clientId,
      scope,
      state,
    });
    if (redirectUri) {
      params.set("redirect_uri", redirectUri);
    }
    window.location.href = `${env.NEXT_PUBLIC_API_BASE_URL}/login/oauth/authorize?${params.toString()}`;
  };

  const handleCancel = () => {
    if (!redirectUri) {
      return;
    }
    const cancelUrl = new URL(redirectUri);
    cancelUrl.searchParams.set("error", "access_denied");
    cancelUrl.searchParams.set("state", state);
    window.location.href = cancelUrl.toString();
  };

  return (
    <div className="min-h-screen bg-[#0d1117] text-[#c9d1d9] font-sans flex items-center justify-center p-6">
      <div className="w-full max-w-[480px]">
        <div className="text-center mb-6">
          <span className="text-5xl text-[#c9d1d9]">⌥</span>
        </div>

        <h1 className="text-center text-2xl font-light mb-6 text-[#c9d1d9]">
          Authorize application
        </h1>

        <div className="bg-[#161b22] border border-[#30363d] rounded-md p-4 mb-4">
          <p className="text-sm text-[#8b949e] mb-2">Application</p>
          <p className="text-base font-semibold text-[#c9d1d9] mb-4 break-all">
            {clientId}
          </p>

          <p className="text-sm text-[#8b949e] mb-3">
            This application is requesting the following permissions:
          </p>

          {scopes.length > 0 ? (
            <ul className="mb-6 space-y-2">
              {scopes.map((s) => (
                <li key={s} className="flex items-start gap-2 text-sm">
                  <Badge
                    variant="outline"
                    className="shrink-0 border-[#30363d] bg-[#0d1117] text-[#58a6ff]"
                  >
                    {s}
                  </Badge>
                  <span className="text-[#c9d1d9]">{formatScope(s)}</span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="mb-6 text-sm text-[#8b949e]">No scopes requested.</p>
          )}

          <div className="flex gap-3">
            <Button
              type="button"
              onClick={handleAuthorize}
              className="flex-1 bg-[#238636] hover:bg-[#2ea043] text-white border border-white/10"
            >
              Authorize
            </Button>
            {redirectUri ? (
              <Button
                type="button"
                variant="outline"
                onClick={handleCancel}
                className="flex-1 border-[#30363d] bg-[#21262d] text-[#c9d1d9] hover:bg-[#30363d] hover:text-[#c9d1d9]"
              >
                Cancel
              </Button>
            ) : (
              <p className="flex-1 text-sm text-[#f85149] self-center" role="alert">
                Cannot cancel without a redirect URI.
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
