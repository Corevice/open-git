"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import {
  BranchProtectionRule,
  deleteBranchProtection,
} from "@/lib/api/branchProtection";

type BranchProtectionListProps = {
  owner: string;
  repo: string;
  rules: BranchProtectionRule[];
};

function yesNo(enabled: boolean): string {
  return enabled ? "Yes" : "No";
}

export function BranchProtectionList({
  owner,
  repo,
  rules,
}: BranchProtectionListProps) {
  const router = useRouter();
  const [deletingPattern, setDeletingPattern] = useState<string | null>(null);

  const handleDelete = async (pattern: string) => {
    const confirmed = window.confirm(
      `Delete branch protection rule for "${pattern}"?`,
    );
    if (!confirmed) {
      return;
    }

    setDeletingPattern(pattern);
    try {
      await deleteBranchProtection(owner, repo, pattern);
      router.refresh();
    } finally {
      setDeletingPattern(null);
    }
  };

  if (rules.length === 0) {
    return (
      <p className="text-sm text-slate-600">No branch protection rules configured.</p>
    );
  }

  return (
    <div className="overflow-x-auto rounded-md border border-slate-200">
      <table className="min-w-full divide-y divide-slate-200 text-sm">
        <thead className="bg-slate-50">
          <tr>
            <th className="px-4 py-3 text-left font-medium text-slate-700">
              Pattern
            </th>
            <th className="px-4 py-3 text-left font-medium text-slate-700">
              Reviews Required
            </th>
            <th className="px-4 py-3 text-left font-medium text-slate-700">
              Force Push
            </th>
            <th className="px-4 py-3 text-left font-medium text-slate-700">
              Delete
            </th>
            <th className="px-4 py-3 text-left font-medium text-slate-700">
              Admin
            </th>
            <th className="px-4 py-3 text-left font-medium text-slate-700">
              Actions
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-200 bg-white">
          {rules.map((rule) => {
            const reviewsRequired =
              rule.required_pull_request_reviews
                ?.required_approving_review_count ?? 0;

            return (
              <tr key={rule.pattern}>
                <td className="px-4 py-3 font-mono">{rule.pattern}</td>
                <td className="px-4 py-3">{reviewsRequired}</td>
                <td className="px-4 py-3">
                  {yesNo(rule.allow_force_pushes.enabled)}
                </td>
                <td className="px-4 py-3">
                  {yesNo(rule.allow_deletions.enabled)}
                </td>
                <td className="px-4 py-3">
                  {yesNo(rule.enforce_admins.enabled)}
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-2">
                    <Link
                      href={`/${owner}/${repo}/settings/branches/${encodeURIComponent(rule.pattern)}/edit`}
                      className="text-indigo-600 hover:underline"
                    >
                      Edit
                    </Link>
                    <Button
                      type="button"
                      variant="destructive"
                      size="sm"
                      disabled={deletingPattern === rule.pattern}
                      onClick={() => handleDelete(rule.pattern)}
                    >
                      Delete
                    </Button>
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
