"use client";

import { use, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  isSuccessStatusCode,
  listDeliveries,
  pingWebhook,
  redeliverDelivery,
  type WebhookDelivery,
} from "@/lib/api/webhook_deliveries";

function formatRelativeTime(dateStr: string | null): string {
  if (!dateStr) return "—";
  const then = new Date(dateStr).getTime();
  if (Number.isNaN(then)) return "—";
  const seconds = Math.floor((Date.now() - then) / 1000);
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} minute${minutes === 1 ? "" : "s"} ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} hour${hours === 1 ? "" : "s"} ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days} day${days === 1 ? "" : "s"} ago`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months} month${months === 1 ? "" : "s"} ago`;
  const years = Math.floor(months / 12);
  return `${years} year${years === 1 ? "" : "s"} ago`;
}

function Toast({
  message,
  variant,
  onDismiss,
}: {
  message: string;
  variant: "success" | "error";
  onDismiss: () => void;
}) {
  useEffect(() => {
    const timer = window.setTimeout(onDismiss, 4000);
    return () => window.clearTimeout(timer);
  }, [onDismiss]);

  return (
    <div
      className={`fixed bottom-6 right-6 z-50 rounded-md border px-4 py-3 text-sm shadow-lg ${
        variant === "success"
          ? "border-[#1f883d] bg-[#dafbe1] text-[#1a7f37]"
          : "border-[#cf222e] bg-[#ffebe9] text-[#cf222e]"
      }`}
      role="status"
    >
      {message}
    </div>
  );
}

export default function WebhookDeliveriesPage({
  params,
}: {
  params: Promise<{ owner: string; repo: string; hookId: string }>;
}) {
  const { owner, repo, hookId: hookIdParam } = use(params);
  const hookId = Number(hookIdParam);

  const [deliveries, setDeliveries] = useState<WebhookDelivery[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pinging, setPinging] = useState(false);
  const [redeliveringId, setRedeliveringId] = useState<string | null>(null);
  const [toast, setToast] = useState<{
    message: string;
    variant: "success" | "error";
  } | null>(null);

  const loadDeliveries = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await listDeliveries(owner, repo, hookId);
      setDeliveries(data);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load deliveries.",
      );
    } finally {
      setLoading(false);
    }
  }, [owner, repo, hookId]);

  useEffect(() => {
    loadDeliveries();
  }, [loadDeliveries]);

  const handlePing = async () => {
    setPinging(true);
    try {
      await pingWebhook(owner, repo, hookId);
      setToast({ message: "Ping sent successfully.", variant: "success" });
      await loadDeliveries();
    } catch (err) {
      setToast({
        message: err instanceof Error ? err.message : "Failed to send ping.",
        variant: "error",
      });
    } finally {
      setPinging(false);
    }
  };

  const handleRedeliver = async (deliveryId: string) => {
    setRedeliveringId(deliveryId);
    try {
      await redeliverDelivery(owner, repo, hookId, deliveryId);
      setToast({ message: "Redelivery queued.", variant: "success" });
      await loadDeliveries();
    } catch (err) {
      setToast({
        message:
          err instanceof Error ? err.message : "Failed to redeliver.",
        variant: "error",
      });
    } finally {
      setRedeliveringId(null);
    }
  };

  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <header className="sticky top-0 z-50 flex h-16 items-center justify-between border-b border-[#d1d9e0] bg-white/85 px-6 backdrop-blur">
        <Link
          href="/dashboard"
          className="flex items-center gap-2 text-lg font-extrabold"
        >
          <span className="text-xl">🐙</span>
          <span>OpenHub</span>
        </Link>
        <Link
          href={`/${owner}/${repo}/settings/hooks/${hookId}/edit`}
          className="rounded-md px-3 py-1.5 text-sm hover:bg-[#f6f8fa]"
        >
          ← Edit webhook
        </Link>
      </header>

      <div className="mx-auto max-w-[1200px] px-6 py-6">
        <div className="mb-4 text-sm text-[#656d76]">
          <Link href={`/${owner}/${repo}`} className="text-[#0969da]">
            {owner}/{repo}
          </Link>{" "}
          /{" "}
          <Link
            href={`/${owner}/${repo}/settings/hooks`}
            className="text-[#0969da]"
          >
            Webhooks
          </Link>{" "}
          /{" "}
          <Link
            href={`/${owner}/${repo}/settings/hooks/${hookId}/edit`}
            className="text-[#0969da]"
          >
            Edit webhook
          </Link>{" "}
          / Recent deliveries
        </div>

        <div className="mb-4 flex items-center justify-between">
          <h1 className="text-2xl font-semibold">Recent deliveries</h1>
          <Button onClick={handlePing} disabled={pinging}>
            {pinging ? "Sending ping…" : "Send ping"}
          </Button>
        </div>

        <p className="mb-6 text-sm text-[#656d76]">
          Recent HTTP POST payloads delivered to this webhook&apos;s configured
          URL.
        </p>

        {loading ? (
          <p className="text-sm text-[#656d76]">Loading…</p>
        ) : error ? (
          <p className="text-sm text-[#cf222e]">{error}</p>
        ) : deliveries.length === 0 ? (
          <div className="rounded-md border border-[#d0d7de] bg-white p-8 text-center">
            <h2 className="text-lg font-semibold">No deliveries yet</h2>
            <p className="mt-2 text-sm text-[#656d76]">
              Deliveries will appear here when events trigger this webhook, or
              after you send a ping.
            </p>
          </div>
        ) : (
          <div className="overflow-hidden rounded-md border border-[#d0d7de] bg-white">
            <table className="w-full table-auto text-sm">
              <thead className="border-b border-[#d0d7de] bg-[#f6f8fa] text-left text-xs uppercase text-[#656d76]">
                <tr>
                  <th className="px-4 py-2">Event</th>
                  <th className="px-4 py-2">Response Code</th>
                  <th className="px-4 py-2">Duration</th>
                  <th className="px-4 py-2">Delivered At</th>
                  <th className="px-4 py-2">Redelivery</th>
                  <th className="px-4 py-2">Actions</th>
                </tr>
              </thead>
              <tbody>
                {deliveries.map((delivery) => {
                  const success = isSuccessStatusCode(delivery.status_code);
                  return (
                    <tr
                      key={delivery.id}
                      className="border-b border-[#eaeef2] last:border-b-0"
                    >
                      <td className="px-4 py-2">
                        <Badge variant="secondary">{delivery.event}</Badge>
                      </td>
                      <td className="px-4 py-2">
                        <span
                          className={
                            delivery.status_code === null
                              ? "text-[#656d76]"
                              : success
                                ? "font-semibold text-[#1a7f37]"
                                : "font-semibold text-[#cf222e]"
                          }
                        >
                          {delivery.status_code ?? "—"}
                        </span>
                      </td>
                      <td className="px-4 py-2 text-xs text-[#656d76]">
                        {delivery.duration_ms !== null
                          ? `${delivery.duration_ms}ms`
                          : "—"}
                      </td>
                      <td className="px-4 py-2 text-xs text-[#656d76]">
                        {formatRelativeTime(delivery.delivered_at)}
                      </td>
                      <td className="px-4 py-2">
                        {delivery.redelivery ? (
                          <Badge variant="outline">Redelivery</Badge>
                        ) : (
                          <span className="text-xs text-[#656d76]">—</span>
                        )}
                      </td>
                      <td className="px-4 py-2">
                        <div className="flex items-center gap-2">
                          <Button variant="outline" size="sm" asChild>
                            <Link
                              href={`/${owner}/${repo}/settings/hooks/${hookId}/deliveries/${delivery.id}`}
                            >
                              View
                            </Link>
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleRedeliver(delivery.id)}
                            disabled={redeliveringId === delivery.id}
                          >
                            {redeliveringId === delivery.id
                              ? "Redelivering…"
                              : "Redeliver"}
                          </Button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {toast && (
        <Toast
          message={toast.message}
          variant={toast.variant}
          onDismiss={() => setToast(null)}
        />
      )}
    </div>
  );
}
