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
  onChange: (branch: string) => void;
}

export default function BranchSelector({ branches, currentBranch, onChange }: Props) {
  return (
    <Select value={currentBranch} onValueChange={onChange}>
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
}: {
  branches: { name: string }[];
  currentBranch: string;
}) {
  const router = useRouter();
  const allowed = branchNamesSet(branches);

  return (
    <BranchSelector
      branches={branches}
      currentBranch={currentBranch}
      onChange={(name) => {
        if (!allowed.has(name)) return;
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
}: {
  owner: string;
  repo: string;
  currentPath: string;
  branches: { name: string }[];
  currentBranch: string;
}) {
  const router = useRouter();
  const allowed = branchNamesSet(branches);

  return (
    <BranchSelector
      branches={branches}
      currentBranch={currentBranch}
      onChange={(name) => {
        if (!allowed.has(name)) return;
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
