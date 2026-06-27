"use client";

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
