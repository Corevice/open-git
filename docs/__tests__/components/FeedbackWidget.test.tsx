import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FeedbackWidget } from "../../components/FeedbackWidget";

describe("FeedbackWidget", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("posts helpful feedback and shows a thank-you message", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 202,
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <FeedbackWidget path="/docs/getting-started" version="latest" />,
    );

    await user.click(screen.getByRole("button", { name: "役に立った" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith("/api/docs/feedback", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          path: "/docs/getting-started",
          helpful: true,
          version: "latest",
        }),
      });
    });

    expect(
      screen.getByText("ごフィードバックありがとうございます。"),
    ).toBeInTheDocument();
  });

  it("does not show error UI when fetch fails", async () => {
    const user = userEvent.setup();
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("network")));

    render(<FeedbackWidget path="/docs/getting-started" version="latest" />);

    await user.click(screen.getByRole("button", { name: "役に立った" }));

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalled();
    });

    expect(screen.queryByText(/error/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/エラー/i)).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "役に立った" }),
    ).toBeInTheDocument();
  });
});
