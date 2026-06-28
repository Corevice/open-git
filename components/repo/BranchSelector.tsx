"use client";

import { useRouter } from "next/navigation";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface Props {
  branches: { name: string }[];
  currentBranch: string;
  onChange?: (branch: string) => void;
  onRefChange?: (ref: string) => void;
}

export default function BranchSelector({
  branches,
  currentBranch,
  onChange,
  onRefChange,
}: Props) {
  const handleChange = (ref: string) => {
    onRefChange?.(ref);
    onChange?.(ref);
  };

  return (
    <Select value={currentBranch} onValueChange={handleChange}>
      <SelectTrigger className="w-[180px]">
        <SelectValue placeholder="Select branch" />
      </SelectTrigger>
      <SelectContent>
        {branches.map((branch) => (
          <SelectItem key={branch.name} value={branch.name}>
            {branch.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function branchNamesSet(branches: { name: string }[]): Set<string> {
  return new Set(branches.map((branch) => branch.name));
}

export function RepoRefSelector({
  branches,
  currentBranch,
  onRefChange,
}: {
  branches: { name: string }[];
  currentBranch: string;
  onRefChange?: (ref: string) => void;
}) {
  const router = useRouter();
  const allowed = branchNamesSet(branches);

  return (
    <BranchSelector
      branches={branches}
      currentBranch={currentBranch}
      onRefChange={onRefChange}
      onChange={(name) => {
        if (!allowed.has(name)) return;
        if (onRefChange) return;
        router.push(`?ref=${encodeURIComponent(name)}`);
      }}
    />
  );
}

export function TreeBranchSelector({
  owner,
  repo,
  currentPath,
  branches,
  currentBranch,
  onRefChange,
}: {
  owner: string;
  repo: string;
  currentPath: string;
  branches: { name: string }[];
  currentBranch: string;
  onRefChange?: (ref: string) => void;
}) {
  const router = useRouter();
  const allowed = branchNamesSet(branches);

  return (
    <BranchSelector
      branches={branches}
      currentBranch={currentBranch}
      onRefChange={onRefChange}
      onChange={(name) => {
        if (!allowed.has(name)) return;
        if (onRefChange) return;
        if (currentPath) {
          const encodedPath = currentPath
            .split("/")
            .map(encodeURIComponent)
            .join("/");
          router.push(`/${owner}/${repo}/tree/${encodeURIComponent(name)}/${encodedPath}`);
          return;
        }
        router.push(`/${owner}/${repo}/tree/${encodeURIComponent(name)}`);
      }}
    />
  );
}
