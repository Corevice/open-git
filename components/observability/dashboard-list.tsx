"use client";

import type { ObservabilityDashboard } from "@/lib/api-types";

interface Props {
  dashboards: ObservabilityDashboard[];
}

export function DashboardList({ dashboards }: Props) {
  if (dashboards.length === 0) {
    return (
      <div className="rounded-md border border-[#d0d7de] bg-white p-6">
        <p className="text-sm text-[#656d76]">No dashboards configured.</p>
      </div>
    );
  }

  const grouped = dashboards.reduce<Record<string, ObservabilityDashboard[]>>(
    (acc, d) => {
      (acc[d.category] ??= []).push(d);
      return acc;
    },
    {},
  );

  return (
    <div className="space-y-6">
      {Object.entries(grouped).map(([cat, items]) => (
        <section key={cat}>
          <h2 className="mb-3 text-lg font-semibold capitalize">{cat}</h2>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {items.map((d) => (
              <div
                key={d.uid}
                className="rounded-md border border-[#d0d7de] bg-white p-4"
              >
                <p className="font-medium text-sm mb-2">{d.title}</p>
                <a
                  href={d.grafana_path}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-[#0969da] text-sm hover:underline"
                >
                  Open in Grafana →
                </a>
              </div>
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}
