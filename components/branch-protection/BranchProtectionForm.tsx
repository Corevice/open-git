"use client";

import { FormEvent, useState } from "react";

import { RequiredStatusChecksSelector } from "@/components/branch-protection/RequiredStatusChecksSelector";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ApiError } from "@/lib/api";
import {
  BranchProtectionRule,
  upsertBranchProtection,
} from "@/lib/api/branchProtection";

type BranchProtectionFormProps = {
  initialValues?: BranchProtectionRule;
  owner: string;
  repo: string;
  onSuccess: () => void;
};

export function BranchProtectionForm({
  initialValues,
  owner,
  repo,
  onSuccess,
}: BranchProtectionFormProps) {
  const isEditMode = initialValues !== undefined;

  const [pattern, setPattern] = useState(initialValues?.pattern ?? "");
  const [requiredApprovingReviewCount, setRequiredApprovingReviewCount] =
    useState(
      initialValues?.required_pull_request_reviews
        ?.required_approving_review_count ?? 0,
    );
  const [dismissStaleReviews, setDismissStaleReviews] = useState(
    initialValues?.required_pull_request_reviews?.dismiss_stale_reviews ??
      false,
  );
  const [requireCodeOwnerReviews, setRequireCodeOwnerReviews] = useState(
    initialValues?.required_pull_request_reviews?.require_code_owner_reviews ??
      false,
  );
  const [statusChecksStrict, setStatusChecksStrict] = useState(
    initialValues?.required_status_checks?.strict ?? false,
  );
  const [statusCheckContexts, setStatusCheckContexts] = useState<string[]>(
    initialValues?.required_status_checks?.contexts ?? [],
  );
  const [enforceAdmins, setEnforceAdmins] = useState(
    initialValues?.enforce_admins.enabled ?? false,
  );
  const [allowForcePushes, setAllowForcePushes] = useState(
    initialValues?.allow_force_pushes.enabled ?? false,
  );
  const [allowDeletions, setAllowDeletions] = useState(
    initialValues?.allow_deletions.enabled ?? false,
  );
  const [requiredLinearHistory, setRequiredLinearHistory] = useState(
    initialValues?.required_linear_history.enabled ?? false,
  );
  const [requiredConversationResolution, setRequiredConversationResolution] =
    useState(initialValues?.required_conversation_resolution.enabled ?? false);

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError(null);

    const trimmedPattern = pattern.trim();
    if (!trimmedPattern) {
      setError("Branch pattern is required.");
      return;
    }

    if (
      requiredApprovingReviewCount < 0 ||
      requiredApprovingReviewCount > 6 ||
      !Number.isInteger(requiredApprovingReviewCount)
    ) {
      setError("Required approving review count must be between 0 and 6.");
      return;
    }

    setSubmitting(true);
    try {
      await upsertBranchProtection(owner, repo, trimmedPattern, {
        required_status_checks:
          statusCheckContexts.length > 0 || statusChecksStrict
            ? {
                strict: statusChecksStrict,
                contexts: statusCheckContexts,
              }
            : null,
        enforce_admins: enforceAdmins,
        required_pull_request_reviews: {
          dismiss_stale_reviews: dismissStaleReviews,
          require_code_owner_reviews: requireCodeOwnerReviews,
          required_approving_review_count: requiredApprovingReviewCount,
        },
        restrictions: null,
        allow_force_pushes: allowForcePushes,
        allow_deletions: allowDeletions,
        required_linear_history: requiredLinearHistory,
        required_conversation_resolution: requiredConversationResolution,
      });
      onSuccess();
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message || "Failed to save branch protection rule.");
      } else {
        setError("Failed to save branch protection rule.");
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <div className="space-y-2">
        <Label htmlFor="pattern">Branch name pattern</Label>
        <Input
          id="pattern"
          value={pattern}
          disabled={isEditMode}
          placeholder="main"
          onChange={(event) => setPattern(event.target.value)}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="required-approving-review-count">
          Required approving review count
        </Label>
        <Input
          id="required-approving-review-count"
          type="number"
          min={0}
          max={6}
          value={requiredApprovingReviewCount}
          onChange={(event) =>
            setRequiredApprovingReviewCount(Number(event.target.value))
          }
        />
      </div>

      <div className="space-y-3">
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={dismissStaleReviews}
            onChange={(event) => setDismissStaleReviews(event.target.checked)}
          />
          Dismiss stale reviews
        </label>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={requireCodeOwnerReviews}
            onChange={(event) =>
              setRequireCodeOwnerReviews(event.target.checked)
            }
          />
          Require code owner reviews
        </label>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={statusChecksStrict}
            onChange={(event) => setStatusChecksStrict(event.target.checked)}
          />
          Require branches to be up to date before merging (strict status checks)
        </label>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={enforceAdmins}
            onChange={(event) => setEnforceAdmins(event.target.checked)}
          />
          Enforce for administrators
        </label>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={allowForcePushes}
            onChange={(event) => setAllowForcePushes(event.target.checked)}
          />
          Allow force pushes
        </label>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={allowDeletions}
            onChange={(event) => setAllowDeletions(event.target.checked)}
          />
          Allow deletions
        </label>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={requiredLinearHistory}
            onChange={(event) => setRequiredLinearHistory(event.target.checked)}
          />
          Require linear history
        </label>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={requiredConversationResolution}
            onChange={(event) =>
              setRequiredConversationResolution(event.target.checked)
            }
          />
          Require conversation resolution before merging
        </label>
      </div>

      <RequiredStatusChecksSelector
        value={statusCheckContexts}
        onChange={setStatusCheckContexts}
      />

      {error && (
        <p className="text-sm text-red-600" role="alert">
          {error}
        </p>
      )}

      <Button type="submit" disabled={submitting}>
        {submitting ? "Saving..." : isEditMode ? "Update rule" : "Create rule"}
      </Button>
    </form>
  );
}
