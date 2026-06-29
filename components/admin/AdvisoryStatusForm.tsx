"use client";

import { FormEvent, useState } from "react";

import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import type { AdvisoryState, DismissedReason } from "@/lib/api-types";

type AdvisoryStatusFormProps = {
  onSubmit: (state: AdvisoryState, reason?: DismissedReason) => void;
};

const STATE_OPTIONS: { value: AdvisoryState; label: string }[] = [
  { value: "open", label: "Open" },
  { value: "acknowledged", label: "Acknowledged" },
  { value: "resolved", label: "Resolved" },
  { value: "dismissed", label: "Dismissed" },
];

const DISMISSED_REASON_OPTIONS: { value: DismissedReason; label: string }[] = [
  { value: "no_bandwidth", label: "No bandwidth" },
  { value: "tolerable_risk", label: "Tolerable risk" },
  { value: "inaccurate", label: "Inaccurate" },
  { value: "not_used", label: "Not used" },
];

export function AdvisoryStatusForm({ onSubmit }: AdvisoryStatusFormProps) {
  const [state, setState] = useState<AdvisoryState>("open");
  const [dismissedReason, setDismissedReason] = useState<
    DismissedReason | ""
  >("");
  const [validationError, setValidationError] = useState<string | null>(null);

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setValidationError(null);

    if (state === "dismissed" && !dismissedReason) {
      setValidationError("Dismissed reason is required when state is dismissed.");
      return;
    }

    if (state === "dismissed") {
      onSubmit(state, dismissedReason as DismissedReason);
      return;
    }

    onSubmit(state);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="advisory-state">State</Label>
        <select
          id="advisory-state"
          value={state}
          onChange={(event) => {
            setState(event.target.value as AdvisoryState);
            setValidationError(null);
          }}
          className="flex h-10 w-full rounded-md border border-[#d0d7de] bg-white px-3 py-2 text-sm text-[#24292f]"
        >
          {STATE_OPTIONS.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      </div>

      {state === "dismissed" && (
        <div className="space-y-2">
          <Label htmlFor="dismissed-reason">Dismissed reason</Label>
          <select
            id="dismissed-reason"
            value={dismissedReason}
            onChange={(event) => {
              setDismissedReason(event.target.value as DismissedReason | "");
              setValidationError(null);
            }}
            className="flex h-10 w-full rounded-md border border-[#d0d7de] bg-white px-3 py-2 text-sm text-[#24292f]"
          >
            <option value="">Select a reason</option>
            {DISMISSED_REASON_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>
      )}

      {validationError && (
        <p role="alert" className="text-sm text-[#cf222e]">
          {validationError}
        </p>
      )}

      <Button type="submit">Update status</Button>
    </form>
  );
}
