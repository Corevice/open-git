"use client";

import Link from "next/link";

export type DocSection = {
  slug: string;
  title: string;
  order?: number;
};

export default function DocSidebar({
  sections,
  currentSlug,
}: {
  sections: DocSection[];
  currentSlug: string;
}) {
  return (
    <nav className="space-y-1">
      {sections.map((s) => (
        <Link
          key={s.slug}
          href={`/docs/contributing/${s.slug}`}
          className={
            s.slug === currentSlug
              ? "font-semibold text-gray-900"
              : "text-gray-600 hover:text-gray-900"
          }
        >
          {s.title}
        </Link>
      ))}
    </nav>
  );
}
