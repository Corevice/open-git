import Link from 'next/link';

type DocSection = {
  slug: string;
  title: string;
  order: number;
};

export default async function ContributingPage() {
  const { sections } = await fetch(
    `${process.env.NEXT_PUBLIC_API_URL}/api/v1/docs/contributing`,
    { next: { revalidate: 3600 } },
  )
    .then((r) => r.json())
    .catch(() => ({ sections: [] as DocSection[] }));

  if (sections.length === 0) {
    return (
      <p>
        CONTRIBUTING.md was not found. Please add a contributing guide to the
        repository.
      </p>
    );
  }

  const sorted = [...sections].sort(
    (a: DocSection, b: DocSection) => a.order - b.order,
  );

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Contributor Guide</h1>
      <ul className="space-y-2">
        {sorted.map((s: DocSection) => (
          <li key={s.slug}>
            <Link
              href={`/docs/contributing/${s.slug}`}
              className="text-blue-600 hover:underline"
            >
              {s.title}
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
