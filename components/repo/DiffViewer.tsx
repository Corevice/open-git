"use client";

import { useState } from "react";

interface DiffLine {
  type: "add" | "remove" | "context" | "hunk";
  content: string;
  oldLine?: number;
  newLine?: number;
}

interface DiffHunk {
  header: string;
  lines: DiffLine[];
}

const HUNK_COLLAPSE_THRESHOLD = 100;

function parsePatch(patch: string): DiffHunk[] {
  const hunks: DiffHunk[] = [];
  let current: DiffHunk | null = null;
  let oldLine = 0;
  let newLine = 0;

  for (const raw of patch.split("\n")) {
    if (raw.startsWith("@@")) {
      current = { header: raw, lines: [] };
      hunks.push(current);
      const match = raw.match(/@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/);
      if (match) {
        oldLine = parseInt(match[1], 10);
        newLine = parseInt(match[2], 10);
      }
      current.lines.push({ type: "hunk", content: raw });
      continue;
    }

    if (!current) continue;

    if (raw.startsWith("+")) {
      current.lines.push({ type: "add", content: raw.slice(1), newLine });
      newLine += 1;
    } else if (raw.startsWith("-")) {
      current.lines.push({ type: "remove", content: raw.slice(1), oldLine });
      oldLine += 1;
    } else if (raw.startsWith(" ") || raw === "") {
      current.lines.push({
        type: "context",
        content: raw.startsWith(" ") ? raw.slice(1) : raw,
        oldLine,
        newLine,
      });
      oldLine += 1;
      newLine += 1;
    }
  }

  return hunks;
}

function lineClass(type: DiffLine["type"]): string {
  if (type === "add") return "bg-green-50";
  if (type === "remove") return "bg-red-50";
  if (type === "hunk") return "bg-[#ddf4ff] text-[#57606a]";
  return "";
}

function HunkBlock({ hunk }: { hunk: DiffHunk }) {
  const contentLines = hunk.lines.filter((l) => l.type !== "hunk");
  const shouldCollapse = contentLines.length > HUNK_COLLAPSE_THRESHOLD;
  const [expanded, setExpanded] = useState(!shouldCollapse);

  const visibleLines = expanded
    ? hunk.lines
    : [hunk.lines[0], ...contentLines.slice(0, 40)];

  const hiddenCount = shouldCollapse && !expanded
    ? contentLines.length - 40
    : 0;

  return (
    <div className="border-t border-[#d0d7de] first:border-t-0">
      <table className="w-full border-collapse font-mono text-xs">
        <tbody>
          {visibleLines.map((line, index) => (
            <tr key={index} className={lineClass(line.type)}>
              <td className="select-none text-right text-[#8c959f] px-2 py-0 w-12 border-r border-[#d0d7de] bg-[#f6f8fa]">
                {line.oldLine ?? ""}
              </td>
              <td className="select-none text-right text-[#8c959f] px-2 py-0 w-12 border-r border-[#d0d7de] bg-[#f6f8fa]">
                {line.newLine ?? ""}
              </td>
              <td className="px-3 py-0 whitespace-pre">
                {line.type === "add" && <span className="text-green-700">+</span>}
                {line.type === "remove" && <span className="text-red-700">-</span>}
                {line.type === "context" && <span className="text-[#57606a]"> </span>}
                {line.type === "hunk" ? line.content : line.content || " "}
              </td>
            </tr>
          ))}
          {hiddenCount > 0 && (
            <tr>
              <td colSpan={3} className="px-3 py-2 bg-[#f6f8fa] border-t border-[#d0d7de]">
                <button
                  type="button"
                  onClick={() => setExpanded(true)}
                  className="text-sm text-[#0969da] hover:underline bg-transparent border-0 cursor-pointer p-0"
                >
                  Show {hiddenCount} more lines
                </button>
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

interface DiffViewerProps {
  filename: string;
  patch?: string;
  additions?: number;
  deletions?: number;
}

export default function DiffViewer({
  filename,
  patch,
  additions = 0,
  deletions = 0,
}: DiffViewerProps) {
  const hunks = patch ? parsePatch(patch) : [];

  return (
    <div className="border border-[#d0d7de] rounded-md overflow-hidden mb-4">
      <div className="px-4 py-2 bg-[#f6f8fa] border-b border-[#d0d7de] flex items-center justify-between gap-3 flex-wrap">
        <span className="font-mono text-sm text-[#24292f]">{filename}</span>
        <span className="text-xs text-[#57606a]">
          {additions > 0 && <span className="text-green-700 font-semibold">+{additions}</span>}
          {additions > 0 && deletions > 0 && " "}
          {deletions > 0 && <span className="text-red-700 font-semibold">-{deletions}</span>}
        </span>
      </div>
      {hunks.length === 0 ? (
        <div className="px-4 py-6 text-sm text-[#57606a]">No diff available.</div>
      ) : (
        hunks.map((hunk, index) => <HunkBlock key={index} hunk={hunk} />)
      )}
    </div>
  );
}
