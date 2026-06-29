"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

import { ApiClient, ApiError } from "@/lib/api";
import type { OrgProfile, Repository, User } from "@/lib/api-types";
import { useAuth } from "@/lib/auth";

type OwnerProfile =
  | { kind: "user"; profile: User }
  | { kind: "org"; profile: OrgProfile };

export default function OwnerPage() {
  const params = useParams<{ owner: string }>();
  const owner = params.owner;
  const { token } = useAuth();

  const apiClient = useMemo(
    () =>
      new ApiClient(
        process.env.NEXT_PUBLIC_API_BASE_URL ??
          process.env.NEXT_PUBLIC_API_URL ??
          "http://localhost:8080",
      ),
    [],
  );

  useEffect(() => {
    if (token) {
      apiClient.setToken(token);
    }
  }, [apiClient, token]);

  const [ownerProfile, setOwnerProfile] = useState<OwnerProfile | null>(null);
  const [repos, setRepos] = useState<Repository[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      if (!owner) {
        return;
      }

      setLoading(true);
      setError(null);

      try {
        let ownerData: OwnerProfile;

        try {
          ownerData = {
            kind: "user",
            profile: await apiClient.users.getByLogin(owner),
          };
        } catch (err) {
          if (err instanceof ApiError && err.status === 404) {
            ownerData = {
              kind: "org",
              profile: await apiClient.orgs.get(owner),
            };
          } else {
            throw err;
          }
        }

        const reposPath =
          ownerData.kind === "user"
            ? `/api/v3/users/${owner}/repos`
            : `/api/v3/orgs/${owner}/repos`;
        const repoList = await apiClient.get<Repository[]>(reposPath);

        if (!cancelled) {
          setOwnerProfile(ownerData);
          setRepos(repoList);
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error ? err.message : "Failed to load profile",
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
  }, [apiClient, owner]);

  if (loading) {
    return (
      <main className="mx-auto max-w-4xl px-4 py-8">
        <p className="text-[color:var(--text-muted)]">Loading profile...</p>
      </main>
    );
  }

  if (error || !ownerProfile) {
    return (
      <main className="mx-auto max-w-4xl px-4 py-8">
        <p className="text-[color:var(--danger)]">
          {error ?? "Profile not found"}
        </p>
      </main>
    );
  }

  const { profile, kind } = ownerProfile;
  const displayName = profile.name ?? profile.login;
  const description =
    kind === "user" ? profile.bio : profile.description;

  return (
    <main className="mx-auto max-w-4xl px-4 py-8">
      <header className="mb-8 border-b border-[color:var(--border)] pb-6">
        <h1 className="text-2xl font-semibold text-[color:var(--text)]">
          {displayName}
        </h1>
        <p className="mt-1 text-[color:var(--text-muted)]">@{profile.login}</p>
        {description ? (
          <p className="mt-4 text-[color:var(--text)]">{description}</p>
        ) : null}
      </header>

      <section>
        <h2 className="mb-4 text-lg font-medium text-[color:var(--text)]">
          Repositories
        </h2>
        {repos.length === 0 ? (
          <p className="text-[color:var(--text-muted)]">No public repositories.</p>
        ) : (
          <ul className="space-y-3">
            {repos.map((repo) => (
              <li
                key={repo.id}
                className="rounded-lg border border-[color:var(--border)] bg-white p-4"
              >
                <Link
                  href={`/${profile.login}/${repo.name}`}
                  className="font-medium text-[color:var(--primary)] hover:underline"
                >
                  {repo.name}
                </Link>
                <p className="mt-1 text-sm text-[color:var(--text-muted)]">
                  {repo.visibility}
                </p>
              </li>
            ))}
          </ul>
        )}
      </section>
    </main>
  );
}
