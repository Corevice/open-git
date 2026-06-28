import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { VersionSelector } from "../../components/VersionSelector";

const push = vi.fn();

vi.mock("next/navigation", () => ({
  usePathname: () => "/latest/en/getting-started",
  useRouter: () => ({ push }),
}));

describe("VersionSelector", () => {
  beforeEach(() => {
    push.mockClear();
  });

  it("renders version options and navigates on change", async () => {
    const user = userEvent.setup();

    render(<VersionSelector />);

    expect(screen.getByRole("option", { name: "latest" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "v1.0" })).toBeInTheDocument();

    await user.selectOptions(
      screen.getByRole("combobox", { name: "Documentation version" }),
      "v1.0",
    );

    expect(push).toHaveBeenCalledWith("/v1.0/en/getting-started");
  });
});
