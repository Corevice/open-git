import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";

import ReviewPanel from "./ReviewPanel";

describe("ReviewPanel", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("submits APPROVE review", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({}),
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<ReviewPanel owner="alice" repo="test" prNumber={1} />);

    fireEvent.change(screen.getByLabelText("Review type"), {
      target: { value: "APPROVE" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Submit review" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/repos/alice/test/pulls/1/reviews",
        expect.objectContaining({
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ event: "APPROVE", body: "" }),
        }),
      );
    });
  });

  it("submits CHANGES_REQUESTED review", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({}),
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<ReviewPanel owner="alice" repo="test" prNumber={1} />);

    fireEvent.change(screen.getByLabelText("Review type"), {
      target: { value: "CHANGES_REQUESTED" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Submit review" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/repos/alice/test/pulls/1/reviews",
        expect.objectContaining({
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ event: "CHANGES_REQUESTED", body: "" }),
        }),
      );
    });
  });
});
