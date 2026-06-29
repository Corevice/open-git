import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import CodeBlock from "../../components/CodeBlock";

describe("CodeBlock", () => {
  it("renders CopyButton and applies language class to code element", () => {
    render(
      <CodeBlock className="language-ts">{`const greeting: string = "hello";`}</CodeBlock>,
    );

    expect(screen.getByRole("button", { name: "コードをコピー" })).toBeInTheDocument();

    const code = document.querySelector("code.language-ts");
    expect(code).toBeInTheDocument();
    expect(code).toHaveClass("language-ts");
  });
});
