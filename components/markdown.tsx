"use client";

import { renderMarkdown } from "@/lib/markdown";

export default function Markdown({ content }: { content: string }) {
  return (
    <div
      className="prose prose-sm max-w-none"
      dangerouslySetInnerHTML={{ __html: renderMarkdown(content) }}
    />
  );
}
