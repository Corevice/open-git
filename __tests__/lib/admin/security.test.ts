import { describe, expect, it } from "vitest";

import {
  buildOrgQueryString,
  datetimeLocalToIso,
  getSeverityBadgeClass,
  isAdminOrgRole,
  maskIpAddress,
  sanitizeAuditSearchPhrase,
  sanitizeOrgLogin,
} from "@/lib/admin/security";

describe("sanitizeOrgLogin", () => {
  it("accepts valid org logins", () => {
    expect(sanitizeOrgLogin("acme-corp")).toBe("acme-corp");
    expect(sanitizeOrgLogin("  acme  ")).toBe("acme");
  });

  it("rejects invalid org logins", () => {
    expect(sanitizeOrgLogin("acme/evil")).toBeNull();
    expect(sanitizeOrgLogin("https://evil.example")).toBeNull();
    expect(sanitizeOrgLogin("")).toBeNull();
    expect(sanitizeOrgLogin(null)).toBeNull();
  });
});

describe("buildOrgQueryString", () => {
  it("builds a safe org query string", () => {
    expect(buildOrgQueryString("acme-corp")).toBe("?org=acme-corp");
    expect(buildOrgQueryString("acme/evil")).toBe("");
  });
});

describe("isAdminOrgRole", () => {
  it("returns true for admin and owner roles", () => {
    expect(isAdminOrgRole("admin")).toBe(true);
    expect(isAdminOrgRole("owner")).toBe(true);
    expect(isAdminOrgRole("OWNER")).toBe(true);
  });

  it("returns false for non-admin roles", () => {
    expect(isAdminOrgRole("member")).toBe(false);
    expect(isAdminOrgRole("read")).toBe(false);
  });
});

describe("sanitizeAuditSearchPhrase", () => {
  it("trims, strips control characters, and caps length", () => {
    expect(sanitizeAuditSearchPhrase("  repo.delete  ")).toBe("repo.delete");
    expect(sanitizeAuditSearchPhrase("a\u0000b")).toBe("ab");
    expect(sanitizeAuditSearchPhrase("x".repeat(300))).toHaveLength(256);
  });
});

describe("datetimeLocalToIso", () => {
  it("converts datetime-local values using local time", () => {
    const iso = datetimeLocalToIso("2024-06-15T10:30");
    expect(iso).toMatch(/^2024-06-15T/);
    expect(iso.endsWith("Z") || iso.includes("+") || iso.includes("-")).toBe(true);
  });

  it("returns empty string for invalid values", () => {
    expect(datetimeLocalToIso("")).toBe("");
    expect(datetimeLocalToIso("not-a-date")).toBe("");
  });
});

describe("maskIpAddress", () => {
  it("masks IPv4 addresses", () => {
    expect(maskIpAddress("192.168.1.42")).toBe("192.168.*.*");
  });

  it("masks IPv6 addresses", () => {
    expect(maskIpAddress("2001:0db8:85a3:0000:0000:8a2e:0370:7334")).toBe(
      "2001:****",
    );
  });

  it("returns em dash for missing values", () => {
    expect(maskIpAddress(null)).toBe("—");
  });
});

describe("getSeverityBadgeClass", () => {
  it("returns a class for known severities", () => {
    expect(getSeverityBadgeClass("critical")).toContain("bg-[#cf222e]");
  });

  it("falls back to low severity styling", () => {
    expect(getSeverityBadgeClass("unknown")).toBe(getSeverityBadgeClass("low"));
  });
});
