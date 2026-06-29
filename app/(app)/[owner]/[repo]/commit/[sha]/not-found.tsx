import Link from "next/link";

export default function CommitNotFound() {
  return (
    <div className="min-h-screen bg-[#f6f8fa] flex items-center justify-center p-6">
      <div className="bg-white border border-[#d0d7de] rounded-lg p-8 max-w-md w-full text-center">
        <h1 className="text-xl font-semibold text-[#24292f] mb-6">
          Commit not found
        </h1>
        <Link
          href="/dashboard"
          className="text-sm text-[#0969da] hover:underline"
        >
          Go to Dashboard
        </Link>
      </div>
    </div>
  );
}
