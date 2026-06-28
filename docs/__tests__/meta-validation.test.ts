import { describe, it, expect } from 'vitest'
import fs from 'fs'
import path from 'path'

const PAGES_DIR = path.join(__dirname, '..', 'pages')

function findMetaFiles(dir: string): string[] {
  const results: string[] = []
  if (!fs.existsSync(dir)) return results

  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      results.push(...findMetaFiles(fullPath))
    } else if (entry.name === '_meta.json') {
      results.push(fullPath)
    }
  }
  return results
}

function findMdxFiles(dir: string): string[] {
  const results: string[] = []
  if (!fs.existsSync(dir)) return results

  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      results.push(...findMdxFiles(fullPath))
    } else if (entry.name.endsWith('.mdx')) {
      results.push(fullPath)
    }
  }
  return results
}

function metaKeyResolvesToFile(pagesDir: string, key: string): boolean {
  const mdxFile = path.join(pagesDir, `${key}.mdx`)
  const indexFile = path.join(pagesDir, key, 'index.mdx')
  const dirPath = path.join(pagesDir, key)

  return (
    fs.existsSync(mdxFile) ||
    fs.existsSync(indexFile) ||
    (fs.existsSync(dirPath) && fs.statSync(dirPath).isDirectory())
  )
}

function hasTitleFrontmatter(content: string): boolean {
  const match = content.match(/^---\s*\n([\s\S]*?)\n---/)
  if (!match) return false
  return /^title\s*:/m.test(match[1])
}

describe('meta validation', () => {
  it('all _meta.json entries resolve to existing files', () => {
    const metaFiles = findMetaFiles(PAGES_DIR)

    for (const metaFile of metaFiles) {
      const dir = path.dirname(metaFile)
      const meta = JSON.parse(fs.readFileSync(metaFile, 'utf-8')) as Record<
        string,
        string
      >

      for (const key of Object.keys(meta)) {
        expect(
          metaKeyResolvesToFile(dir, key),
          `_meta.json key "${key}" in ${path.relative(PAGES_DIR, metaFile)} does not resolve to a file`
        ).toBe(true)
      }
    }
  })

  it('all MDX files have title frontmatter', () => {
    const mdxFiles = findMdxFiles(PAGES_DIR)

    for (const mdxFile of mdxFiles) {
      const content = fs.readFileSync(mdxFile, 'utf-8')
      expect(
        hasTitleFrontmatter(content),
        `${path.relative(PAGES_DIR, mdxFile)} is missing title frontmatter`
      ).toBe(true)
    }
  })

  it('finds at least 10 MDX pages', () => {
    const mdxFiles = findMdxFiles(PAGES_DIR)
    expect(mdxFiles.length).toBeGreaterThanOrEqual(10)
  })
})
