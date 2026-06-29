import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useAuth } from "@/components/providers/auth-provider";
import type { Viewer } from "@/types/viewer";

export function useViewer() {
  const { token } = useAuth();
  const { data, isLoading, isError } = useQuery({
    queryKey: ["viewer"],
    queryFn: () => apiClient.get<Viewer>("/api/v3/user", { token: token! }),
    enabled: !!token,
  });
  return { viewer: data ?? null, isLoading, isError };
}
