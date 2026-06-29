import { describe, expect, it } from "vitest";

import { BRANDING } from "@/lib/branding";

const FORBIDDEN_TERMS = ["GitHub", "Octocat"];

describe("BRANDING", () => {
  it("exports appName as a non-empty string up to 50 characters", () => {
    expect(typeof BRANDING.appName).toBe("string");
    expect(BRANDING.appName.length).toBeGreaterThan(0);
    expect(BRANDING.appName.length).toBeLessThanOrEqual(50);
  });

  it("does not include forbidden standalone brand terms", () => {
    for (const term of FORBIDDEN_TERMS) {
      expect(BRANDING.appName).not.toBe(term);
      expect(BRANDING.appName.toLowerCase()).not.toContain(term.toLowerCase());
    }
  });
});
