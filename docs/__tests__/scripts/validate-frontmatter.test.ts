import fs from "fs";
import os from "os";
import path from "path";
import { afterEach, describe, expect, it } from "vitest";
import {
  validateFrontmatter,
  validateFrontmatterOrThrow,
} from "../../scripts/validate-frontmatter";

const tempDirs: string[] = [];

function createTempPagesDir(): string {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "mdx-frontmatter-"));
  tempDirs.push(dir);
  return dir;
}

afterEach(() => {
  for (const dir of tempDirs.splice(0)) {
    fs.rmSync(dir, { recursive: true, force: true });
  }
});

describe("validateFrontmatter", () => {
  it("returns an error when title is missing", () => {
    const pagesDir = createTempPagesDir();
    fs.writeFileSync(
      path.join(pagesDir, "missing-title.mdx"),
      "---\ndescription: test\n---\n# Hello",
    );

    const result = validateFrontmatter(pagesDir);

    expect(result.errors).toHaveLength(1);
    expect(result.errors[0]?.message).toMatch(/title/i);
    expect(() => validateFrontmatterOrThrow(pagesDir)).toThrow(/title/i);
  });

  it("succeeds when title is present", () => {
    const pagesDir = createTempPagesDir();
    fs.writeFileSync(
      path.join(pagesDir, "valid.mdx"),
      "---\ntitle: Hello World\n---\n# Hello",
    );

    const result = validateFrontmatter(pagesDir);

    expect(result.errors).toHaveLength(0);
    expect(result.successCount).toBe(1);
    expect(validateFrontmatterOrThrow(pagesDir).successCount).toBe(1);
  });

  it("flags draft pages in production mode", () => {
    const pagesDir = createTempPagesDir();
    fs.writeFileSync(
      path.join(pagesDir, "draft.mdx"),
      "---\ntitle: Draft Page\ndraft: true\n---\n# Draft",
    );

    const devResult = validateFrontmatter(pagesDir, { production: false });
    const prodResult = validateFrontmatter(pagesDir, { production: true });

    expect(devResult.errors).toHaveLength(0);
    expect(devResult.successCount).toBe(1);
    expect(prodResult.errors).toHaveLength(1);
    expect(prodResult.errors[0]?.message).toMatch(/draft/i);
    expect(() =>
      validateFrontmatterOrThrow(pagesDir, { production: true }),
    ).toThrow(/draft/i);
  });
});
