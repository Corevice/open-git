import { render, screen } from "@testing-library/react";
import { LanguageSelector } from "../../components/LanguageSelector";

vi.mock("next/navigation", () => ({
  usePathname: () => "/latest/ja/getting-started",
}));

vi.mock("next/link", () => ({
  default: ({
    href,
    children,
    ...props
  }: {
    href: string;
    children: React.ReactNode;
  }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

describe("LanguageSelector", () => {
  it("renders ja and en locale links", () => {
    render(<LanguageSelector />);

    expect(screen.getByRole("link", { name: "ja" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "en" })).toBeInTheDocument();
  });
});
