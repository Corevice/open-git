// @vitest-environment node

import { readFileSync } from "fs";
import { join } from "path";
import { describe, expect, it } from "vitest";

describe("404 page", () => {
  it("includes title frontmatter and a link back to the docs home", () => {
    const content = readFileSync(
      join(__dirname, "../../pages/404.mdx"),
      "utf-8",
    );

    expect(content).toMatch(/title:\s*ページが見つかりません/);
    expect(content).toMatch(/\]\(\/docs[^)]*\)/);
  });
});
