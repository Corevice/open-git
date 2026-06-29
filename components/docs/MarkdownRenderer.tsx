"use client";

import { useRef, useEffect } from "react";
import { renderMarkdown } from "@/lib/markdown";
import Prism from "prismjs";
import "prismjs/components/prism-bash";
import "prismjs/components/prism-typescript";
import "prismjs/components/prism-go";

export default function MarkdownRenderer({ content }: { content: string }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!ref.current) return;
    ref.current.querySelectorAll("h2, h3").forEach((el) => {
      if (!el.id)
        el.id = el.textContent!
          .toLowerCase()
          .replace(/[^a-z0-9]+/g, "-")
          .replace(/(^-|-$)/g, "");
    });
    ref.current.querySelectorAll("a[href]").forEach((el) => {
      const a = el as HTMLAnchorElement;
      if (a.href.startsWith("http")) {
        a.target = "_blank";
        a.rel = "noopener noreferrer";
      }
    });
    Prism.highlightAllUnder(ref.current);
    ref.current.querySelectorAll("pre").forEach((pre) => {
      if (pre.querySelector(".copy-btn")) return;
      const btn = document.createElement("button");
      btn.className = "copy-btn";
      btn.textContent = "Copy";
      btn.onclick = () => navigator.clipboard.writeText(pre.textContent ?? "");
      pre.style.position = "relative";
      pre.appendChild(btn);
    });
  }, [content]);
  return (
    <div
      ref={ref}
      className="prose prose-sm max-w-none"
      dangerouslySetInnerHTML={{ __html: renderMarkdown(content) }}
    />
  );
}
