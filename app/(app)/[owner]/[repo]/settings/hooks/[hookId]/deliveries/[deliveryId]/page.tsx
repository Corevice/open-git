"use client";

import { use, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  formatBody,
  formatHeaders,
  getDelivery,
  isSuccessStatusCode,
  redeliverDelivery,
  type WebhookDeliveryDetail,
} from "@/lib/api/webhook_deliveries";

function CodeBlock({ content }: { content: string }) {
  if (!content) {
    return (
      <p className="text-sm italic text-[#656d76]">No content recorded.</p>
    );
  }

  return (
    <pre className="overflow-x-auto rounded-md border border-[#d0d7de] bg-[#f6f8fa] p-4 text-xs font-mono whitespace-pre-wrap break-all">
      {content}
    </pre>
  );
}

export default function WebhookDeliveryDetailPage({
  params,
}: {
  params: Promise<{
    owner: string;
    repo: string;
    hookId: string;
    deliveryId: string;
  }>;
}) {
  const { owner, repo, hookId: hookIdParam, deliveryId } = use(params);
  const hookId = Number(hookIdParam);
  const router = useRouter();

  const [delivery, setDelivery] = useState<WebhookDeliveryDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [redelivering, setRedelivering] = useState(false);
  const [redeliverError, setRedeliverError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      setError(null);
      try {
        const data = await getDelivery(owner, repo, hookId, deliveryId);
        if (!cancelled) setDelivery(data);
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error ? err.message : "Failed to load delivery.",
          );
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => {
      cancelled = true;
    };
  }, [owner, repo, hookId, deliveryId]);

  const handleRedeliver = async () => {
    setRedelivering(true);
    setRedeliverError(null);
    try {
      await redeliverDelivery(owner, repo, hookId, deliveryId);
      router.push(`/${owner}/${repo}/settings/hooks/${hookId}/deliveries`);
    } catch (err) {
      setRedeliverError(
        err instanceof Error ? err.message : "Failed to redeliver.",
      );
    } finally {
      setRedelivering(false);
    }
  };

  const requestBody = formatBody(delivery?.request_body);
  const responseBody = formatBody(delivery?.response_body);

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
          href={`/${owner}/${repo}/settings/hooks/${hookId}/deliveries`}
          className="rounded-md px-3 py-1.5 text-sm hover:bg-[#f6f8fa]"
        >
          ← Deliveries
        </Link>
      </header>

      <div className="mx-auto max-w-4xl px-6 py-6">
        <div className="mb-4 text-sm text-[#656d76]">
          <Link href={`/${owner}/${repo}`} className="text-[#0969da]">
            {owner}/{repo}
          </Link>{" "}
          /{" "}
          <Link
            href={`/${owner}/${repo}/settings/hooks/${hookId}/deliveries`}
            className="text-[#0969da]"
          >
            Recent deliveries
          </Link>{" "}
          / {deliveryId.slice(0, 8)}…
        </div>

        {loading ? (
          <p className="text-sm text-[#656d76]">Loading…</p>
        ) : error || !delivery ? (
          <div>
            <p className="text-sm text-[#cf222e]">
              {error ?? "Delivery not found."}
            </p>
            <Button variant="outline" asChild className="mt-4">
              <Link
                href={`/${owner}/${repo}/settings/hooks/${hookId}/deliveries`}
              >
                Back to deliveries
              </Link>
            </Button>
          </div>
        ) : (
          <>
            <div className="mb-6 flex items-center justify-between">
              <div>
                <h1 className="text-2xl font-semibold">Delivery detail</h1>
                <div className="mt-2 flex flex-wrap items-center gap-2">
                  <Badge variant="secondary">{delivery.event}</Badge>
                  {delivery.redelivery && (
                    <Badge variant="outline">Redelivery</Badge>
                  )}
                </div>
              </div>
              <Button onClick={handleRedeliver} disabled={redelivering}>
                {redelivering ? "Redelivering…" : "Redeliver"}
              </Button>
            </div>

            {redeliverError && (
              <p className="mb-4 text-sm text-[#cf222e]" role="alert">
                {redeliverError}
              </p>
            )}

            <section className="mb-6 rounded-md border border-[#d0d7de] bg-white p-6">
              <h2 className="mb-4 text-lg font-semibold">Request</h2>
              <h3 className="mb-2 text-sm font-semibold text-[#656d76]">
                Headers
              </h3>
              <CodeBlock content={formatHeaders(delivery.request_headers)} />
              <h3 className="mb-2 mt-4 text-sm font-semibold text-[#656d76]">
                Body
              </h3>
              <CodeBlock content={requestBody.text} />
            </section>

            <section className="rounded-md border border-[#d0d7de] bg-white p-6">
              <h2 className="mb-4 text-lg font-semibold">Response</h2>
              <div className="mb-4 text-sm">
                <span className="font-semibold text-[#656d76]">
                  HTTP status:{" "}
                </span>
                <span
                  className={
                    delivery.status_code === null
                      ? "text-[#656d76]"
                      : isSuccessStatusCode(delivery.status_code)
                        ? "font-semibold text-[#1a7f37]"
                        : "font-semibold text-[#cf222e]"
                  }
                >
                  {delivery.status_code ?? "—"}
                </span>
                {delivery.duration_ms !== null && (
                  <span className="ml-4 text-[#656d76]">
                    ({delivery.duration_ms}ms)
                  </span>
                )}
              </div>
              <h3 className="mb-2 text-sm font-semibold text-[#656d76]">
                Headers
              </h3>
              <CodeBlock content={formatHeaders(delivery.response_headers)} />
              <h3 className="mb-2 mt-4 text-sm font-semibold text-[#656d76]">
                Body
              </h3>
              <CodeBlock content={responseBody.text} />
              {responseBody.truncated && (
                <p className="mt-2 text-xs text-[#656d76]">
                  Response body truncated at 64KB.
                </p>
              )}
            </section>
          </>
        )}
      </div>
    </div>
  );
}
