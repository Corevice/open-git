import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { UntranslatedBanner } from "../../components/UntranslatedBanner";

describe("UntranslatedBanner", () => {
  it("shows banner for en locale on ja-only pages and hides it when dismissed", async () => {
    const user = userEvent.setup();

    render(<UntranslatedBanner locale="en" pageLang="ja" />);

    expect(
      screen.getByText("このページはまだ日本語のみ提供されています。"),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "閉じる" }));

    expect(
      screen.queryByText("このページはまだ日本語のみ提供されています。"),
    ).not.toBeInTheDocument();
  });
});
