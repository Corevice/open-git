interface EmptyRepoStateProps {
  cloneUrl: string;
}

export default function EmptyRepoState({ cloneUrl }: EmptyRepoStateProps) {
  return (
    <div className="p-8 text-center">
      <h2 className="text-lg font-semibold text-[#24292f] m-0 mb-4">No code yet</h2>
      <p className="text-sm text-[#57606a] mb-4">
        Get started by cloning this repository and pushing your first commit.
      </p>
      <code className="block rounded-md border border-[#d0d7de] bg-[#f6f8fa] px-4 py-2 font-mono text-sm text-[#24292f] mb-6">
        {cloneUrl}
      </code>
      <ol className="list-decimal text-left text-sm text-[#57606a] space-y-2 max-w-md mx-auto pl-5">
        <li>
          <code className="font-mono text-[#24292f]">git init</code>
        </li>
        <li>
          <code className="font-mono text-[#24292f]">
            git remote add origin {cloneUrl}
          </code>
        </li>
        <li>
          <code className="font-mono text-[#24292f]">
            git push -u origin main
          </code>
        </li>
      </ol>
    </div>
  );
}
