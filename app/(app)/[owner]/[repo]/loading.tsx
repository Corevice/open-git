export default function RepoLoading() {
  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <div className="animate-pulse">
        <div className="h-16 bg-gray-200" />

        <div className="bg-white border-b border-[#d0d7de] py-4">
          <div className="max-w-[1280px] mx-auto px-6">
            <div className="bg-gray-200 rounded h-6 w-64 mb-4" />
            <div className="flex gap-2">
              <div className="bg-gray-200 rounded h-4 w-16" />
              <div className="bg-gray-200 rounded h-4 w-24" />
              <div className="bg-gray-200 rounded h-4 w-20" />
            </div>
          </div>
        </div>

        <div className="max-w-[1280px] mx-auto px-6 py-6">
          <div className="bg-white border border-[#d0d7de] rounded-lg overflow-hidden">
            <div className="p-3 border-b border-[#d0d7de] flex items-center gap-3">
              <div className="bg-gray-200 rounded h-4 w-32" />
              <div className="bg-gray-200 rounded h-4 w-20" />
            </div>
            <div className="p-2 space-y-2">
              {Array.from({ length: 7 }).map((_, i) => (
                <div
                  key={i}
                  className="bg-gray-200 rounded h-4"
                  style={{ width: `${60 + (i % 3) * 10}%` }}
                />
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
