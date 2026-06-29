import { beforeEach, describe, expect, it, vi } from "vitest";
import { ZodError } from "zod";

describe("env validation", () => {
  beforeEach(() => {
    vi.resetModules();
    vi.unstubAllEnvs();
    // The test runner provides a default NEXT_PUBLIC_API_BASE_URL; clear it so
    // each case controls the environment explicitly.
    delete process.env.NEXT_PUBLIC_API_BASE_URL;
    delete process.env.NEXT_PUBLIC_APP_VERSION;
  });

  it("parses a valid NEXT_PUBLIC_API_BASE_URL", async () => {
    vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "http://localhost:8080");

    const { env } = await import("@/lib/env");

    expect(env.NEXT_PUBLIC_API_BASE_URL).toBe("http://localhost:8080");
  });

  it("throws ZodError for a non-URL value", async () => {
    vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "not-a-url");

    await expect(import("@/lib/env")).rejects.toThrow(ZodError);
  });

  it("throws ZodError when NEXT_PUBLIC_API_BASE_URL is missing", async () => {
    await expect(import("@/lib/env")).rejects.toThrow(ZodError);
  });
});
