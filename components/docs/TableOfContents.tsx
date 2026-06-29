"use client";

import { useEffect, useMemo, useState } from "react";

function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
}

export default function TableOfContents({
  content,
  className,
}: {
  content: string;
  className?: string;
}) {
  const headings = useMemo(() => {
    const matches = content.match(/^#{2,3} .+/gm);
    if (!matches) return [];
    return matches.map((line) => {
      const level = line.startsWith("###") ? 3 : 2;
      const text = line.replace(/^#{2,3}\s+/, "");
      return { level, text, id: slugify(text) };
    });
  }, [content]);

  const [activeId, setActiveId] = useState<string | null>(
    headings[0]?.id ?? null,
  );

  useEffect(() => {
    if (headings.length === 0) return;

    const elements = headings
      .map((h) => document.getElementById(h.id))
      .filter((el): el is HTMLElement => el != null);

    if (elements.length === 0) return;

    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((e) => e.isIntersecting)
          .sort(
            (a, b) => a.boundingClientRect.top - b.boundingClientRect.top,
          );
        if (visible.length > 0) {
          setActiveId(visible[0].target.id);
        }
      },
      { rootMargin: "-20% 0px -70% 0px", threshold: 0 },
    );

    elements.forEach((el) => observer.observe(el));
    return () => observer.disconnect();
  }, [headings]);

  if (headings.length === 0) return null;

  return (
    <nav className={`sticky top-4 ${className ?? ""}`}>
      <ul className="space-y-1 text-sm">
        {headings.map((h) => (
          <li key={h.id} className={h.level === 3 ? "ml-4" : ""}>
            <a
              href={`#${h.id}`}
              className={
                activeId === h.id
                  ? "font-semibold text-blue-600"
                  : "text-gray-600 hover:text-blue-600"
              }
            >
              {h.text}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  );
}
