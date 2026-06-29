import Link from "next/link";

export default function RepoNotFound() {
  return (
    <div className="min-h-screen bg-[#f6f8fa] flex items-center justify-center p-6">
      <div className="bg-white border border-[#d0d7de] rounded-lg p-8 max-w-md w-full text-center">
        <h1 className="text-xl font-semibold text-[#24292f] mb-2">
          Repository not found
        </h1>
        <p className="text-sm text-[#656d76] mb-6">
          The repository you are looking for does not exist or you do not have
          access to it.
        </p>
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
