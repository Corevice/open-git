"use client";

import { useEffect, useMemo } from "react";
import Prism from "prismjs";
import "prismjs/components/prism-bash";
import "prismjs/components/prism-css";
import "prismjs/components/prism-go";
import "prismjs/components/prism-javascript";
import "prismjs/components/prism-json";
import "prismjs/components/prism-markdown";
import "prismjs/components/prism-python";
import "prismjs/components/prism-sql";
import "prismjs/components/prism-tsx";
import "prismjs/components/prism-typescript";
import "prismjs/components/prism-yaml";
import "prismjs/themes/prism.css";

interface BlobViewerProps {
  content: string;
  filename: string;
  binary?: boolean;
  truncated?: boolean;
  rawUrl?: string;
}

function languageFromFilename(filename: string): string {
  const ext = filename.split(".").pop()?.toLowerCase() ?? "";
  const map: Record<string, string> = {
    ts: "typescript",
    tsx: "tsx",
    js: "javascript",
    jsx: "javascript",
    json: "json",
    md: "markdown",
    markdown: "markdown",
    py: "python",
    go: "go",
    sh: "bash",
    bash: "bash",
    yml: "yaml",
    yaml: "yaml",
    css: "css",
    sql: "sql",
  };
  return map[ext] ?? "plaintext";
}

export default function BlobViewer({
  content,
  filename,
  binary = false,
  truncated = false,
  rawUrl,
}: BlobViewerProps) {
  const language = useMemo(() => languageFromFilename(filename), [filename]);
  const lines = useMemo(() => content.split("\n"), [content]);

  const highlightedLines = useMemo(() => {
    if (binary || truncated) return [];
    const grammar = Prism.languages[language] ?? Prism.languages.plaintext;
    return lines.map((line) =>
      Prism.highlight(line, grammar, language),
    );
  }, [binary, truncated, language, lines]);

  useEffect(() => {
    if (!binary && !truncated) {
      Prism.highlightAll();
    }
  }, [binary, truncated, highlightedLines]);

  if (binary) {
    return (
      <div className="p-8 text-center text-sm text-[#57606a] bg-[#f6f8fa] border-t border-[#d0d7de]">
        Binary file not shown
      </div>
    );
  }

  return (
    <div>
      {truncated && (
        <div className="px-4 py-3 bg-yellow-50 border-b border-yellow-200 text-sm text-yellow-900">
          File is too large to display
          {rawUrl && (
            <>
              {" — "}
              <a href={rawUrl} className="text-[#0969da] hover:underline font-medium">
                View raw
              </a>
            </>
          )}
        </div>
      )}
      {!truncated && (
        <div className="overflow-x-auto">
          <table className="w-full border-collapse font-mono text-[13px] leading-5">
            <tbody>
              {lines.map((line, index) => (
                <tr key={index} className="hover:bg-[#f6f8fa]">
                  <td className="select-none text-right text-[#8c959f] px-3 py-0 align-top border-r border-[#eaeef2] bg-[#f6f8fa] w-12">
                    {index + 1}
                  </td>
                  <td className="px-4 py-0 align-top whitespace-pre text-[#24292f]">
                    <code
                      className={`language-${language}`}
                      dangerouslySetInnerHTML={{ __html: highlightedLines[index] || " " }}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
