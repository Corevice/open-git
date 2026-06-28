type EmptyStateProps = {
  title: string;
  description?: string;
};

export function EmptyState({ title, description }: EmptyStateProps) {
  return (
    <div className="px-4 py-8 text-center text-[#656d76]" data-testid="empty-state">
      <p className="font-semibold text-[#1f2328]">{title}</p>
      {description ? <p className="text-sm mt-1">{description}</p> : null}
    </div>
  );
}
