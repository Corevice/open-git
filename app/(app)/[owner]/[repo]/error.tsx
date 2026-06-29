"use client";

export default function RepoError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="min-h-screen bg-[#f6f8fa] flex items-center justify-center p-6">
      <div className="bg-white border border-[#d0d7de] rounded-lg p-8 max-w-md w-full text-center">
        <div className="text-4xl mb-4">⚠️</div>
        <h2 className="text-lg font-semibold text-[#24292f] mb-2">
          Something went wrong
        </h2>
        <p className="text-xs font-mono text-[#656d76] mb-6 break-all">
          {error.message}
        </p>
        <button
          onClick={reset}
          className="px-4 py-2 text-sm bg-[color:var(--primary)] text-white rounded-md hover:opacity-90"
        >
          Try again
        </button>
      </div>
    </div>
  );
}
