export default function EditOnRepoLink({ editUrl }: { editUrl: string }) {
  if (!editUrl) return null;
  return (
    <a
      href={editUrl}
      target="_blank"
      rel="noopener noreferrer"
      className="text-xs text-gray-500 hover:text-blue-600"
    >
      Edit this page
    </a>
  );
}
