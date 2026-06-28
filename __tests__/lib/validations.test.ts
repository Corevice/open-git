import { describe, expect, it } from "vitest";

import { commentSchema, issueSchema, repoNameSchema } from "@/lib/validations";

describe("issueSchema", () => {
  it("rejects empty title", () => {
    expect(issueSchema.safeParse({ title: "" }).success).toBe(false);
  });

  it("rejects title of 257 characters", () => {
    expect(issueSchema.safeParse({ title: "a".repeat(257) }).success).toBe(
      false,
    );
  });

  it("accepts title of 1 character", () => {
    expect(issueSchema.safeParse({ title: "a" }).success).toBe(true);
  });

  it("allows body to be optional", () => {
    expect(issueSchema.safeParse({ title: "a" }).success).toBe(true);
    expect(issueSchema.safeParse({ title: "a", body: "details" }).success).toBe(
      true,
    );
  });
});

describe("commentSchema", () => {
  it("rejects empty string body", () => {
    expect(commentSchema.safeParse({ body: "" }).success).toBe(false);
  });

  it("rejects whitespace-only body", () => {
    expect(commentSchema.safeParse({ body: "   " }).success).toBe(false);
  });

  it("accepts non-empty body", () => {
    expect(commentSchema.safeParse({ body: "Looks good" }).success).toBe(true);
  });
});

describe("repoNameSchema", () => {
  it("rejects names starting with a dot", () => {
    expect(repoNameSchema.safeParse(".foo").success).toBe(false);
  });

  it("rejects names ending with a dot", () => {
    expect(repoNameSchema.safeParse("foo.").success).toBe(false);
  });

  it("rejects reserved name settings", () => {
    expect(repoNameSchema.safeParse("settings").success).toBe(false);
  });

  it("accepts my-repo", () => {
    expect(repoNameSchema.safeParse("my-repo").success).toBe(true);
  });

  it("accepts my-repo-123", () => {
    expect(repoNameSchema.safeParse("my-repo-123").success).toBe(true);
  });

  it("accepts repo.name", () => {
    expect(repoNameSchema.safeParse("repo.name").success).toBe(true);
  });
});
