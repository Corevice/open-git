import { describe, expect, it } from "vitest";

import { ApiError } from "../api";
import { isSecretValidationError } from "./secrets";

describe("isSecretValidationError", () => {
  it("returns true for 422 ApiError with fieldErrors", () => {
    const error = new ApiError(422, "Validation failed") as ApiError & {
      fieldErrors: Record<string, string>;
    };
    error.fieldErrors = { name: "名前は必須です" };

    expect(isSecretValidationError(error)).toBe(true);
  });

  it("returns false for non-422 ApiError", () => {
    expect(isSecretValidationError(new ApiError(401, "Unauthorized"))).toBe(
      false,
    );
  });

  it("returns false for non-ApiError values", () => {
    expect(isSecretValidationError(new Error("boom"))).toBe(false);
    expect(isSecretValidationError(null)).toBe(false);
  });
});
