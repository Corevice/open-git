"use client";

import { FormEvent, useEffect, useState } from "react";
import {
  validateSecretName,
  type SecretVisibility,
} from "@/lib/api/secrets";

export interface SecretFormProps {
  onSubmit: (
    name: string,
    value: string,
    visibility?: SecretVisibility,
  ) => Promise<void>;
  existingName?: string;
  initialVisibility?: SecretVisibility;
  showVisibility?: boolean;
  submitting: boolean;
  formError: string | null;
  fieldErrors: Record<string, string>;
  onCancel: () => void;
}

function isSecretVisibility(value: string): value is SecretVisibility {
  return value === "all" || value === "private" || value === "selected";
}

function visibilityLabel(visibility: SecretVisibility): string {
  switch (visibility) {
    case "all":
      return "All repositories";
    case "private":
      return "Private repositories";
    case "selected":
      return "Selected repositories";
  }
}

export function SecretForm({
  onSubmit,
  existingName,
  initialVisibility = "all",
  showVisibility = false,
  submitting,
  formError,
  fieldErrors,
  onCancel,
}: SecretFormProps) {
  const isEditing = existingName !== undefined;
  const [name, setName] = useState(existingName ?? "");
  const [value, setValue] = useState("");
  const [visibility, setVisibility] =
    useState<SecretVisibility>(initialVisibility);
  const trimmedValue = value.trim();
  const trimmedName = name.trim();
  const nameValidationError = isEditing ? null : validateSecretName(trimmedName);
  const canSubmit = isEditing
    ? true
    : trimmedName.length > 0 && trimmedValue.length > 0 && !nameValidationError;

  useEffect(() => {
    setName(existingName ?? "");
  }, [existingName]);

  useEffect(() => {
    setVisibility(initialVisibility);
  }, [initialVisibility]);

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    const secretName = existingName ?? trimmedName;

    if (!isEditing) {
      const validationError = validateSecretName(secretName);
      if (validationError || !trimmedValue) {
        return;
      }
    }

    try {
      await onSubmit(
        secretName,
        trimmedValue,
        showVisibility ? visibility : undefined,
      );
      setValue("");
    } catch {
      // Preserve the value so the user can retry after an error.
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {formError && (
        <p className="text-sm text-destructive" role="alert">
          {formError}
        </p>
      )}

      <div>
        <label
          htmlFor="secret-name"
          className="mb-1 block text-sm font-medium"
        >
          Name {!isEditing && <span className="text-destructive">*</span>}
        </label>
        <input
          id="secret-name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          disabled={isEditing}
          placeholder="MY_SECRET"
          className="flex h-10 w-full rounded-md border border-slate-300 bg-white px-3 py-2 font-mono text-sm ring-offset-white placeholder:text-slate-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
          aria-invalid={fieldErrors.name || nameValidationError ? true : undefined}
        />
        {(fieldErrors.name || nameValidationError) && (
          <p className="mt-1 text-sm text-destructive">
            {fieldErrors.name ?? nameValidationError}
          </p>
        )}
      </div>

      <div>
        <label
          htmlFor="secret-value"
          className="mb-1 block text-sm font-medium"
        >
          Value {!isEditing && <span className="text-destructive">*</span>}
        </label>
        <textarea
          id="secret-value"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder={isEditing ? "Enter a new value" : "Secret value"}
          rows={4}
          className="flex w-full rounded-md border border-slate-300 bg-white px-3 py-2 font-mono text-sm ring-offset-white placeholder:text-slate-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
          aria-invalid={fieldErrors.value ? true : undefined}
          autoComplete="new-password"
        />
        {fieldErrors.value && (
          <p className="mt-1 text-sm text-destructive">{fieldErrors.value}</p>
        )}
      </div>

      {showVisibility && (
        <div>
          <label
            htmlFor="secret-visibility"
            className="mb-1 block text-sm font-medium"
          >
            Repository access
          </label>
          <select
            id="secret-visibility"
            value={visibility}
            onChange={(event) => {
              const next = event.target.value;
              if (isSecretVisibility(next)) {
                setVisibility(next);
              }
            }}
            className="flex h-10 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:ring-offset-2"
            aria-invalid={fieldErrors.visibility ? true : undefined}
          >
            <option value="all">{visibilityLabel("all")}</option>
            <option value="private">{visibilityLabel("private")}</option>
            <option value="selected">{visibilityLabel("selected")}</option>
          </select>
          {fieldErrors.visibility && (
            <p className="mt-1 text-sm text-destructive">
              {fieldErrors.visibility}
            </p>
          )}
        </div>
      )}

      <div className="flex justify-end gap-2">
        <button
          type="button"
          onClick={onCancel}
          className="inline-flex h-10 items-center justify-center rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
          disabled={submitting}
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={submitting || !canSubmit}
          className="inline-flex h-10 items-center justify-center rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {submitting
            ? isEditing
              ? "Updating…"
              : "Creating…"
            : isEditing
              ? "Update secret"
              : "Add secret"}
        </button>
      </div>
    </form>
  );
}
