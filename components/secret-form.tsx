"use client";

import { FormEvent, useEffect, useState } from "react";

export interface SecretFormProps {
  onSubmit: (name: string, value: string) => Promise<void>;
  existingName?: string;
  submitting: boolean;
  formError: string | null;
  fieldErrors: Record<string, string>;
  onCancel: () => void;
}

export function SecretForm({
  onSubmit,
  existingName,
  submitting,
  formError,
  fieldErrors,
  onCancel,
}: SecretFormProps) {
  const isEditing = existingName !== undefined;
  const [name, setName] = useState(existingName ?? "");
  const [value, setValue] = useState("");

  useEffect(() => {
    setName(existingName ?? "");
  }, [existingName]);

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    const secretName = existingName ?? name.trim();
    await onSubmit(secretName, value);
    setValue("");
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="space-y-4 rounded-md border border-[#d0d7de] bg-white p-5"
    >
      <div>
        <label
          htmlFor="secret-name"
          className="mb-1.5 block text-sm font-semibold"
        >
          Name <span className="text-[#cf222e]">*</span>
        </label>
        <input
          id="secret-name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          disabled={isEditing}
          placeholder="MY_SECRET"
          className="w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm font-mono disabled:cursor-not-allowed disabled:bg-[#f6f8fa] disabled:text-[#656d76]"
          required
        />
        {fieldErrors["name"] && (
          <p className="text-red-500 text-xs mt-1">{fieldErrors["name"]}</p>
        )}
      </div>

      <div>
        <label
          htmlFor="secret-value"
          className="mb-1.5 block text-sm font-semibold"
        >
          Value {!isEditing && <span className="text-[#cf222e]">*</span>}
        </label>
        <textarea
          id="secret-value"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder={isEditing ? "Leave blank to keep current value" : "Secret value"}
          rows={4}
          className="w-full rounded-md border border-[#d0d7de] px-3 py-2 text-sm font-mono"
          required={!isEditing}
          autoComplete="new-password"
        />
        {fieldErrors["value"] && (
          <p className="text-red-500 text-xs mt-1">{fieldErrors["value"]}</p>
        )}
      </div>

      {formError && (
        <p className="text-sm text-[#cf222e]" role="alert">
          {formError}
        </p>
      )}

      <div className="flex items-center justify-end gap-4">
        <button
          type="button"
          onClick={onCancel}
          className="text-sm text-[#0969da] hover:underline"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={submitting}
          className="rounded-md bg-[#1f883d] px-4 py-2 text-sm font-semibold text-white hover:bg-[#1a7f37] disabled:cursor-not-allowed disabled:opacity-50"
        >
          {submitting
            ? "Saving…"
            : existingName
              ? "Update secret"
              : "Add secret"}
        </button>
      </div>
    </form>
  );
}
