import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

import { ArtifactList, formatSize } from "@/components/actions/ArtifactList";
import type { Artifact } from "@/lib/api/actions";

const defaultProps = {
  owner: "acme",
  repo: "demo",
  loading: false,
};

function makeArtifact(overrides: Partial<Artifact> = {}): Artifact {
  return {
    id: 1,
    name: "build-output",
    size_in_bytes: 1024,
    expired: false,
    created_at: "2024-01-01T00:00:00Z",
    expires_at: "2024-02-01T00:00:00Z",
    ...overrides,
  };
}

describe("ArtifactList", () => {
  it("expired artifact has aria-disabled=true on download link", () => {
    render(
      <ArtifactList
        {...defaultProps}
        artifacts={[makeArtifact({ id: 99, expired: true })]}
      />,
    );

    const link = screen.getByRole("link", { name: "Download" });
    expect(link).toHaveAttribute("aria-disabled", "true");
    expect(link).toHaveAttribute("title", "Artifact has expired");
  });

  it("non-expired artifact renders clickable link", () => {
    render(
      <ArtifactList
        {...defaultProps}
        artifacts={[makeArtifact({ id: 42, expired: false })]}
      />,
    );

    const link = screen.getByRole("link", { name: "Download" });
    expect(link).toHaveAttribute(
      "href",
      "/api/repos/acme/demo/actions/artifacts/42/zip",
    );
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).not.toHaveAttribute("aria-disabled");
  });

  it("formats artifact sizes as KB and MB", () => {
    render(
      <ArtifactList
        {...defaultProps}
        artifacts={[
          makeArtifact({ id: 1, size_in_bytes: 1536 }),
          makeArtifact({ id: 2, name: "large-artifact", size_in_bytes: 2097152 }),
        ]}
      />,
    );

    expect(screen.getByText(/1\.5 KB/)).toBeInTheDocument();
    expect(screen.getByText(/2\.0 MB/)).toBeInTheDocument();
  });
});

describe("formatSize", () => {
  it("formats kilobytes and megabytes", () => {
    expect(formatSize(1536)).toBe("1.5 KB");
    expect(formatSize(2097152)).toBe("2.0 MB");
  });
});
