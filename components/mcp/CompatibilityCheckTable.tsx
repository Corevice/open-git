"use client";

import { Fragment, useState } from "react";

import { VerificationResultDetail } from "@/components/mcp/VerificationResultDetail";
import { Badge } from "@/components/ui/badge";
import {
  TableRoot as Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { MCPVerificationCheck } from "@/lib/api-client";
import { cn } from "@/lib/utils";

interface CompatibilityCheckTableProps {
  checks: MCPVerificationCheck[];
  overallStatus: string | null;
}

function overallStatusClass(status: string | null): string {
  switch (status) {
    case "compatible":
      return "bg-[#dafbe1] text-[#1a7f37] border-[#4ac26b]";
    case "partial":
      return "bg-[#fff8c5] text-[#9a6700] border-[#d4a72c]";
    case "incompatible":
      return "bg-[#ffebe9] text-[#cf222e] border-[#ff8182]";
    default:
      return "bg-[#f6f8fa] text-[#656d76] border-[#d0d7de]";
  }
}

function categoryClass(category: MCPVerificationCheck["category"]): string {
  switch (category) {
    case "graphql":
      return "bg-[#ddf4ff] text-[#0969da] border-[#54aeff]";
    case "rest":
      return "bg-[#dafbe1] text-[#1a7f37] border-[#4ac26b]";
    case "auth":
      return "bg-[#fbefff] text-[#8250df] border-[#d8b9ff]";
  }
}

function checkStatusClass(status: MCPVerificationCheck["status"]): string {
  switch (status) {
    case "pass":
      return "bg-[#dafbe1] text-[#1a7f37] border-[#4ac26b]";
    case "fail":
      return "bg-[#ffebe9] text-[#cf222e] border-[#ff8182]";
    case "skip":
      return "bg-[#f6f8fa] text-[#656d76] border-[#d0d7de]";
  }
}

export function CompatibilityCheckTable({
  checks,
  overallStatus,
}: CompatibilityCheckTableProps) {
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const toggleRow = (checkId: string) => {
    setExpandedId((current) => (current === checkId ? null : checkId));
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <span className="text-sm font-medium text-muted-foreground">
          Overall status:
        </span>
        <Badge
          variant="outline"
          className={cn(
            "capitalize",
            overallStatusClass(overallStatus),
          )}
        >
          {overallStatus ?? "unknown"}
        </Badge>
      </div>

      <div className="overflow-hidden rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Check ID</TableHead>
              <TableHead>Category</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Duration</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {checks.map((check) => {
              const isExpanded = expandedId === check.id;

              return (
                <Fragment key={check.id}>
                  <TableRow
                    className={check.status === "fail" ? "cursor-pointer" : undefined}
                    onClick={() => toggleRow(check.id)}
                  >
                    <TableCell className="font-mono text-xs">{check.id}</TableCell>
                    <TableCell>
                      <Badge
                        variant="outline"
                        className={cn("capitalize", categoryClass(check.category))}
                      >
                        {check.category}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant="outline"
                        className={cn("capitalize", checkStatusClass(check.status))}
                      >
                        {check.status}
                      </Badge>
                    </TableCell>
                    <TableCell>{check.duration_ms}ms</TableCell>
                  </TableRow>
                  {isExpanded && check.status === "fail" && (
                    <TableRow>
                      <TableCell colSpan={4} className="bg-muted/30">
                        <VerificationResultDetail check={check} />
                      </TableCell>
                    </TableRow>
                  )}
                </Fragment>
              );
            })}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
