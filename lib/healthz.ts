export interface HealthResponse {
  status: string;
  version: string;
  time: string;
}

export async function checkHealth(): Promise<HealthResponse | null> {
  try {
    const res = await fetch(
      new URL("/healthz", process.env.NEXT_PUBLIC_API_BASE_URL!).toString(),
    );
    if (!res.ok) return null;
    return res.json() as Promise<HealthResponse>;
  } catch {
    return null;
  }
}
