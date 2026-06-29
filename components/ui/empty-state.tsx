import Link from "next/link";

import { Button } from "@/components/ui/button";

interface EmptyStateProps {
  title: string;
  description?: string;
  action?: {
    label: string;
    href: string;
  };
}

export function EmptyState({ title, description, action }: EmptyStateProps) {
  return (
    <div
      className="flex flex-col items-center py-16 text-center gap-4"
      data-testid="empty-state"
    >
      <h3>{title}</h3>
      {description ? <p>{description}</p> : null}
      {action && (
        <Button variant="outline" asChild>
          <Link href={action.href}>{action.label}</Link>
        </Button>
      )}
    </div>
  );
}
