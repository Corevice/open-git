import Markdown from "@/components/markdown";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { cn } from "@/lib/utils";

type ReviewThreadProps = {
  reviewer: { login: string; avatar_url?: string };
  state: "approved" | "changes_requested" | "commented";
  body: string;
  createdAt: string;
};

const stateConfig: Record<
  ReviewThreadProps["state"],
  { label: string; className: string }
> = {
  approved: { label: "Approved", className: "bg-[#dafbe1] text-[#1a7f37]" },
  changes_requested: {
    label: "Changes requested",
    className: "bg-[#fff8c5] text-[#9a6700]",
  },
  commented: { label: "Commented", className: "bg-[#eaeef2] text-[#656d76]" },
};

export function ReviewThread({
  reviewer,
  state,
  body,
  createdAt,
}: ReviewThreadProps) {
  const config = stateConfig[state];

  return (
    <div className="flex gap-3 border-t border-[#d0d7de] px-4 py-3">
      <Avatar className="h-8 w-8">
        {reviewer.avatar_url && (
          <AvatarImage src={reviewer.avatar_url} alt={reviewer.login} />
        )}
        <AvatarFallback>{reviewer.login.slice(0, 2).toUpperCase()}</AvatarFallback>
      </Avatar>
      <div className="min-w-0 flex-1">
        <div className="mb-2 flex flex-wrap items-center gap-2 text-sm">
          <span className="font-semibold text-[#1f2328]">{reviewer.login}</span>
          <span
            className={cn(
              "inline-block rounded-full px-2 py-0.5 text-xs font-semibold",
              config.className,
            )}
          >
            {config.label}
          </span>
          <span className="text-xs text-[#656d76]">{createdAt}</span>
        </div>
        <Markdown content={body} />
      </div>
    </div>
  );
}
