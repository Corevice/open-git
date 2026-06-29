import { notFound } from 'next/navigation';

import EditOnRepoLink from '@/components/docs/EditOnRepoLink';
import MarkdownRenderer from '@/components/docs/MarkdownRenderer';
import TableOfContents from '@/components/docs/TableOfContents';

type Props = { params: Promise<{ slug: string }> };

export default async function DocSectionPage({ params }: Props) {
  const { slug } = await params;
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_API_URL}/api/v1/docs/contributing/${slug}`,
    { next: { revalidate: 3600 } },
  );
  if (res.status === 404) notFound();
  if (!res.ok) throw new Error('Failed to fetch doc section');
  const section = await res.json();
  return (
    <div className="flex gap-8">
      <article className="flex-1 max-w-3xl">
        <h1 className="text-2xl font-bold mb-4">{section.title}</h1>
        <MarkdownRenderer content={section.content_markdown} />
        <div className="mt-4">
          <EditOnRepoLink editUrl={section.edit_url} />
        </div>
      </article>
      <aside className="w-48 shrink-0 sticky top-8 self-start">
        <TableOfContents content={section.content_markdown} />
      </aside>
    </div>
  );
}

export async function generateStaticParams() {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? '';
  const res = await fetch(`${apiUrl}/api/v1/docs/contributing`).catch(
    () => null,
  );
  if (!res?.ok) return [];
  const { sections } = await res.json();
  return sections.map((s: { slug: string }) => ({ slug: s.slug }));
}
