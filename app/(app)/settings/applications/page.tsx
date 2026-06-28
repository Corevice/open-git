"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { API_TOKEN_KEY, ApiClient } from "@/lib/api";
import type { OAuthAuthorizationInfo } from "@/lib/api-types";
import { useAuth } from "@/lib/auth";

interface AuthorizedApp {
  client_id: string;
  name: string;
  scopes: string[];
}

function createApiClient(): ApiClient {
  const baseURL =
    process.env.NEXT_PUBLIC_API_BASE_URL ??
    process.env.NEXT_PUBLIC_API_URL ??
    "http://localhost:8080";
  const client = new ApiClient(baseURL);
  if (typeof window !== "undefined") {
    const storedToken = localStorage.getItem(API_TOKEN_KEY);
    if (storedToken) {
      client.setToken(storedToken);
    }
  }
  return client;
}

function toAuthorizedApp(item: OAuthAuthorizationInfo): AuthorizedApp {
  return {
    client_id: item.oauth_app_id,
    name: item.app_name,
    scopes: item.granted_scopes,
  };
}

export default function ApplicationsSettingsPage() {
  const { isAuthenticated } = useAuth();
  const router = useRouter();
  const api = useMemo(() => createApiClient(), []);

  const [apps, setApps] = useState<AuthorizedApp[]>([]);
  const [revoking, setRevoking] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push("/login");
    }
  }, [isAuthenticated, router]);

  useEffect(() => {
    if (!isAuthenticated) {
      return;
    }

    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(null);
      try {
        const list = await api.userAuthorizations.list();
        if (!cancelled) {
          setApps(list.map(toAuthorizedApp));
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error
              ? err.message
              : "Failed to load authorized OAuth applications.",
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
  }, [api, isAuthenticated]);

  const appToRevoke = apps.find((app) => app.client_id === revoking) ?? null;

  const handleConfirmRevoke = async () => {
    if (!revoking || submitting) {
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      await api.userAuthorizations.revoke(revoking);
      setApps((prev) => prev.filter((app) => app.client_id !== revoking));
      setRevoking(null);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to revoke OAuth application access.",
      );
    } finally {
      setSubmitting(false);
    }
  };

  if (!isAuthenticated) {
    return null;
  }

  return (
    <main className="mx-auto max-w-[960px] px-6 py-8">
      <div className="mb-6 border-b border-[#d1d9e0] pb-4">
        <h1 className="mb-2 text-2xl font-semibold">Applications</h1>
        <p className="text-sm text-[#59636e]">
          Manage OAuth applications that have access to your account.
        </p>
      </div>

      <nav className="mb-6 flex gap-4 text-sm">
        <Link href="/settings/profile" className="text-[#0969da] hover:underline">
          Profile
        </Link>
        <Link href="/settings/tokens" className="text-[#0969da] hover:underline">
          Personal Access Tokens
        </Link>
        <span className="font-semibold text-[#0969da]">Applications</span>
      </nav>

      <section>
        <h2 className="mb-4 text-lg font-semibold">Authorized OAuth Applications</h2>

        {error && (
          <p className="mb-4 text-sm text-[#cf222e]" role="alert">
            {error}
          </p>
        )}

        {loading ? (
          <p className="text-sm text-[#59636e]">Loading…</p>
        ) : apps.length === 0 ? (
          <p className="text-sm text-[#59636e]">
            You have not authorized any OAuth applications.
          </p>
        ) : (
          <div className="overflow-x-auto rounded-lg border border-[#d1d9e0]">
            <table className="min-w-full text-left text-sm">
              <thead className="border-b border-[#d1d9e0] bg-[#f6f8fa]">
                <tr>
                  <th className="px-4 py-3 font-semibold">Application</th>
                  <th className="px-4 py-3 font-semibold">Scopes</th>
                  <th className="px-4 py-3 font-semibold">Actions</th>
                </tr>
              </thead>
              <tbody>
                {apps.map((app) => (
                  <tr
                    key={app.client_id}
                    className="border-b border-[#d1d9e0] last:border-b-0"
                  >
                    <td className="px-4 py-3 font-medium">{app.name}</td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {app.scopes.map((scope) => (
                          <Badge key={scope} variant="secondary">
                            {scope}
                          </Badge>
                        ))}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <Button
                        type="button"
                        variant="destructive"
                        size="sm"
                        onClick={() => setRevoking(app.client_id)}
                        disabled={revoking === app.client_id && submitting}
                      >
                        Revoke
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <Dialog
        open={revoking !== null}
        onOpenChange={(open) => {
          if (!open && !submitting) {
            setRevoking(null);
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Revoke application access</DialogTitle>
            <DialogDescription>
              {appToRevoke
                ? `Are you sure you want to revoke access for ${appToRevoke.name}? All tokens issued to this app will be invalidated.`
                : "Are you sure you want to revoke access for this application? All tokens issued to this app will be invalidated."}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setRevoking(null)}
              disabled={submitting}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={handleConfirmRevoke}
              disabled={submitting}
            >
              {submitting ? "Revoking…" : "Revoke access"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </main>
  );
}
