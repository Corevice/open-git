import Link from "next/link";

export default function NotFound() {
  return (
    <div className="flex min-h-[50vh] flex-col items-center justify-center gap-4 p-6">
      <h2 className="text-xl font-semibold">Repository not found</h2>
      <Link href="/dashboard" className="text-indigo-600 hover:underline">
        Back to dashboard
      </Link>
    </div>
  );
}
