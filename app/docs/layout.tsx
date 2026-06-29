import DocSidebar from '@/components/docs/DocSidebar';

export default async function DocsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { sections } = await fetch(
    `${process.env.NEXT_PUBLIC_API_URL}/api/v1/docs/contributing`,
    { next: { revalidate: 3600 } },
  )
    .then((r) => r.json())
    .catch(() => ({ sections: [] }));

  return (
    <div className="flex min-h-screen">
      <aside className="w-64 shrink-0">
        <DocSidebar sections={sections} currentSlug="" />
      </aside>
      <main className="flex-1 p-6">{children}</main>
    </div>
  );
}
