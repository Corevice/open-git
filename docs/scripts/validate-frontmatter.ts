import fs from "fs";
import path from "path";
import matter from "gray-matter";

export interface ValidationError {
  file: string;
  message: string;
}

export interface ValidationResult {
  errors: ValidationError[];
  successCount: number;
}

export interface ValidateOptions {
  production?: boolean;
}

function isProduction(options: ValidateOptions): boolean {
  if (options.production !== undefined) {
    return options.production;
  }
  return process.env.NODE_ENV === "production";
}

function walkMdxFiles(dir: string, files: string[] = []): string[] {
  if (!fs.existsSync(dir)) {
    return files;
  }

  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      walkMdxFiles(fullPath, files);
    } else if (entry.isFile() && entry.name.endsWith(".mdx")) {
      files.push(fullPath);
    }
  }

  return files;
}

export function validateFrontmatter(
  pagesDir: string,
  options: ValidateOptions = {},
): ValidationResult {
  const production = isProduction(options);
  const errors: ValidationError[] = [];
  let successCount = 0;

  for (const filePath of walkMdxFiles(pagesDir)) {
    const content = fs.readFileSync(filePath, "utf8");
    const { data } = matter(content);

    const title = data.title;
    if (typeof title !== "string" || title.trim() === "") {
      errors.push({
        file: filePath,
        message: 'Missing required "title" in frontmatter',
      });
      continue;
    }

    if (production && data.draft === true) {
      errors.push({
        file: filePath,
        message: "Draft pages are not allowed in production builds",
      });
      continue;
    }

    successCount++;
  }

  return { errors, successCount };
}

export function validateFrontmatterOrThrow(
  pagesDir: string,
  options: ValidateOptions = {},
): ValidationResult {
  const result = validateFrontmatter(pagesDir, options);
  if (result.errors.length > 0) {
    const details = result.errors
      .map((error) => `${error.file}: ${error.message}`)
      .join("\n");
    throw new Error(`Frontmatter validation failed:\n${details}`);
  }
  return result;
}

function main(): void {
  const pagesDir = path.join(process.cwd(), "pages");
  const result = validateFrontmatter(pagesDir);

  if (result.errors.length > 0) {
    console.error("Frontmatter validation failed:");
    for (const error of result.errors) {
      console.error(`  ${error.file}: ${error.message}`);
    }
    process.exit(1);
  }

  console.log(
    `Frontmatter validation passed: ${result.successCount} file(s) checked`,
  );
}

const isMainModule =
  process.argv[1] !== undefined &&
  path.resolve(process.argv[1]) === path.resolve(__filename);

if (isMainModule) {
  main();
}
