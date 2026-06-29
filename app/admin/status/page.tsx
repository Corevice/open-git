"use client";

import { useEffect, useState } from "react";

import { AdminStatusCard } from "@/components/AdminStatusCard";
import { SystemMetrics } from "@/components/SystemMetrics";
import { useAuth } from "@/lib/auth";

const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  process.env.NEXT_PUBLIC_API_URL ??
  "";

type HealthData = Record<string, string>;

type AdminStatusData = {
  queue_depth?: number;
  db_connections?: number;
};

interface UserSession {
  is_admin: boolean;
}

function mapHealthStatus(value: string): "ok" | "error" | "unknown" {
  if (value === "ok") return "ok";
  if (value === "error") return "error";
  return "unknown";
}

async function fetchJson<T>(path: string, token?: string): Promise<T | null> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}${path}`, {
    headers,
    cache: "no-store",
  });

  if (!response.ok) {
    return null;
  }

  const body = (await response.json()) as { data?: T };
  return body.data ?? (body as T);
}

async function fetchUserSession(token: string): Promise<UserSession | null> {
  const response = await fetch(`${API_BASE}/api/v3/user`, {
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${token}`,
    },
    cache: "no-store",
  });

  if (!response.ok) {
    return null;
  }

  const user = (await response.json()) as {
    is_admin?: boolean;
    site_admin?: boolean;
  };

  return {
    is_admin: user.is_admin ?? user.site_admin ?? false,
  };
}

export default function AdminStatusPage() {
  const { token } = useAuth();
  const [health, setHealth] = useState<HealthData | null>(null);
  const [adminStatus, setAdminStatus] = useState<AdminStatusData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);

      const healthData = await fetchJson<HealthData>("/api/v1/health");
      if (!cancelled) {
        setHealth(healthData);
      }

      if (token) {
        const session = await fetchUserSession(token);
        if (session?.is_admin) {
          const [adminData] = await Promise.all([
            fetchJson<AdminStatusData>("/api/v1/admin/status", token),
            fetchJson<{ version: string; commit: string; buildDate: string }>(
              "/api/v1/version",
              token,
            ),
          ]);
          if (!cancelled) {
            setAdminStatus(adminData);
          }
        }
      }

      if (!cancelled) {
        setLoading(false);
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [token]);

  const adminMetrics: Record<string, number> = {};
  if (adminStatus?.queue_depth != null) {
    adminMetrics.queue_depth = adminStatus.queue_depth;
  }
  if (adminStatus?.db_connections != null) {
    adminMetrics.db_connections = adminStatus.db_connections;
  }

  return (
    <div className="mx-auto max-w-4xl px-6 py-8">
      <h1 className="mb-6 text-2xl font-semibold">System Status</h1>

      {loading && <p className="text-sm text-gray-500">Loading status…</p>}

      {health && (
        <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Object.entries(health).map(([name, status]) => (
            <AdminStatusCard
              key={name}
              name={name}
              status={mapHealthStatus(status)}
            />
          ))}
        </div>
      )}

      {adminStatus && Object.keys(adminMetrics).length > 0 && (
        <section>
          <h2 className="mb-4 text-lg font-medium">System Metrics</h2>
          <SystemMetrics metrics={adminMetrics} />
        </section>
      )}
    </div>
  );
}
