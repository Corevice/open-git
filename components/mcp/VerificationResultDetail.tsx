import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import type { MCPVerificationCheck } from "@/lib/api-client";

interface VerificationResultDetailProps {
  check: MCPVerificationCheck;
}

export function VerificationResultDetail({
  check,
}: VerificationResultDetailProps) {
  if (check.expected === null && check.actual === null) {
    return (
      <p className="text-sm text-muted-foreground">Check was skipped</p>
    );
  }

  return (
    <div className="space-y-4">
      {check.error !== null && (
        <Alert variant="destructive">
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{check.error}</AlertDescription>
        </Alert>
      )}
      <div className="grid gap-4 md:grid-cols-2">
        <div>
          <p className="mb-2 text-xs font-semibold uppercase text-muted-foreground">
            Expected
          </p>
          <pre className="overflow-x-auto rounded-md border bg-muted/50 p-3 text-xs">
            {JSON.stringify(check.expected, null, 2)}
          </pre>
        </div>
        <div>
          <p className="mb-2 text-xs font-semibold uppercase text-muted-foreground">
            Actual
          </p>
          <pre className="overflow-x-auto rounded-md border bg-muted/50 p-3 text-xs">
            {JSON.stringify(check.actual, null, 2)}
          </pre>
        </div>
      </div>
    </div>
  );
}
