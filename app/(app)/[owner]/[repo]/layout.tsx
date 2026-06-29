import ErrorBoundary from "@/components/ErrorBoundary";

export default function RepoLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <ErrorBoundary>{children}</ErrorBoundary>;
}
