import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import CopyButton from "../../components/CopyButton";

describe("CopyButton", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("copies code to clipboard and shows confirmation text", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    // userEvent.setup() installs its own navigator.clipboard stub, so define the
    // spy afterwards to ensure the component writes through to this mock.
    const user = userEvent.setup();
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });

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
