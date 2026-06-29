import Link from "next/link";
import type { CommitEntry } from "@/types/repo";

interface CommitRowProps {
  entry: CommitEntry;
  owner: string;
  repo: string;
}

function truncateMessage(message: string): string {
  const firstLine = message.split("\n")[0];
  if (firstLine.length <= 72) {
    return firstLine;
  }
  return `${firstLine.slice(0, 72)}…`;
}

function formatRelativeDate(dateStr: string): string {
  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) {
    return dateStr;
  }

  const diffSeconds = Math.round((date.getTime() - Date.now()) / 1000);
  const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });

  const divisions: { amount: number; unit: Intl.RelativeTimeFormatUnit }[] = [
    { amount: 60, unit: "second" },
    { amount: 60, unit: "minute" },
    { amount: 24, unit: "hour" },
    { amount: 7, unit: "day" },
    { amount: 4.34524, unit: "week" },
    { amount: 12, unit: "month" },
    { amount: Infinity, unit: "year" },
  ];

  let duration = diffSeconds;
  for (const { amount, unit } of divisions) {
    if (Math.abs(duration) < amount) {
      return rtf.format(duration, unit);
    }
    duration /= amount;
  }

  return rtf.format(duration, "year");
}

export default function CommitRow({ entry, owner, repo }: CommitRowProps) {
  const shortSha = entry.sha.slice(0, 7);
  const message = truncateMessage(entry.commit.message);
  const authorName = entry.commit.author.name;
  const relativeDate = formatRelativeDate(entry.commit.author.date);

  return (
    <li>
      <Link href={`/${owner}/${repo}/commit/${entry.sha}`}>
        <code>{shortSha}</code>
        <span>{message}</span>
        <span>{authorName}</span>
        <span>{relativeDate}</span>
      </Link>
    </li>
  );
}
