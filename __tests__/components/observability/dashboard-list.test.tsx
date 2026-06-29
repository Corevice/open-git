import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

import { DashboardList } from "@/components/observability/dashboard-list";
import type { ObservabilityDashboard } from "@/lib/api-types";

describe("DashboardList", () => {
  it("renders empty state when no dashboards", () => {
    render(<DashboardList dashboards={[]} />);

    expect(screen.getByText("No dashboards configured.")).toBeInTheDocument();
  });

  it("renders dashboard titles", () => {
    const dashboards: ObservabilityDashboard[] = [
      {
        uid: "system-overview",
        title: "System概況",
        category: "system",
        grafana_path: "/d/system-overview",
      },
      {
        uid: "git-operations",
        title: "Git操作",
        category: "git",
        grafana_path: "/d/git-operations",
      },
    ];

    render(<DashboardList dashboards={dashboards} />);

    expect(screen.getByText("System概況")).toBeInTheDocument();
    expect(screen.getByText("Git操作")).toBeInTheDocument();
  });

  it("renders Open in Grafana links with correct href", () => {
    const dashboards: ObservabilityDashboard[] = [
      {
        uid: "system-overview",
        title: "System概況",
        category: "system",
        grafana_path: "/d/system-overview",
      },
    ];

    render(<DashboardList dashboards={dashboards} />);

    const link = screen.getByRole("link", { name: /Open in Grafana/ });
    expect(link).toHaveAttribute("href", "/d/system-overview");
    expect(link).toHaveAttribute("target", "_blank");
  });
});
