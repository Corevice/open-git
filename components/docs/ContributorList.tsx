"use client";

import { useState } from "react";

export type Contributor = {
  login: string;
  id: number;
  avatar_url: string;
  contributions: number;
  type: string;
};

function initials(login: string): string {
  return login.slice(0, 2).toUpperCase();
}

function ContributorItem({ contributor }: { contributor: Contributor }) {
  const [avatarError, setAvatarError] = useState(!contributor.avatar_url);

  return (
    <div className="flex flex-col items-center gap-1">
      {avatarError ? (
        <span className="flex h-10 w-10 items-center justify-center rounded-full bg-gray-200 text-xs font-semibold text-gray-700">
          {initials(contributor.login)}
        </span>
      ) : (
        <img
          src={contributor.avatar_url}
          alt={contributor.login}
          className="h-10 w-10 rounded-full"
          onError={() => setAvatarError(true)}
        />
      )}
      <span className="text-sm">{contributor.login}</span>
      <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs">
        {contributor.contributions}
      </span>
    </div>
  );
}

export default function ContributorList({
  contributors,
}: {
  contributors: Contributor[];
}) {
  return (
    <div className="flex flex-wrap gap-4">
      {contributors.map((c) => (
        <ContributorItem key={c.id} contributor={c} />
      ))}
    </div>
  );
}
