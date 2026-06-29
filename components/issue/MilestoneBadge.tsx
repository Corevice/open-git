type MilestoneBadgeProps = {
  title: string;
  openCount: number;
  closedCount: number;
};

export function MilestoneBadge({ title, openCount, closedCount }: MilestoneBadgeProps) {
  const total = openCount + closedCount;
  const value = total > 0 ? closedCount / total : 0;

  return (
    <span className="inline-flex items-center gap-2 text-xs text-[#656d76]">
      <span>{title}</span>
      <progress
        className="h-2 w-16"
        value={value}
        max={1}
        aria-label={`${title} progress`}
      />
    </span>
  );
}
