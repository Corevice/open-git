import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import CopyButton from "../../components/CopyButton";

describe("CopyButton", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("copies code to clipboard and shows confirmation text", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", {
      clipboard: { writeText },
    });

    const user = userEvent.setup();
    render(<CopyButton code="hello" />);

    expect(screen.getByRole("button", { name: "コードをコピー" })).toHaveTextContent(
      "コピー",
    );

    await user.click(screen.getByRole("button", { name: "コードをコピー" }));

    expect(writeText).toHaveBeenCalledWith("hello");
    expect(screen.getByRole("button", { name: "コードをコピーしました" })).toHaveTextContent(
      "コピーしました！",
    );
  });
});
