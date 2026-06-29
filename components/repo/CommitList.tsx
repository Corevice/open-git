import type { CommitEntry } from "@/types/repo";
import CommitRow from "@/components/repo/CommitRow";

interface CommitListProps {
  commits: CommitEntry[];
  owner: string;
  repo: string;
}

export default function CommitList({ commits, owner, repo }: CommitListProps) {
  if (commits.length === 0) {
    return <p>No commits yet.</p>;
  }

  return (
    <ul>
      {commits.map((entry) => (
        <CommitRow key={entry.sha} entry={entry} owner={owner} repo={repo} />
      ))}
    </ul>
  );
}
