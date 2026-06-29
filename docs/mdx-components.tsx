import type { MDXComponents } from "mdx/types";
import type { ReactNode } from "react";
import CodeBlock from "./components/CodeBlock";

type CalloutVariant = "note" | "warning" | "danger";

const calloutBorderColors: Record<CalloutVariant, string> = {
  note: "#0969da",
  warning: "#bf8700",
  danger: "#cf222e",
};

type CalloutProps = {
  type?: CalloutVariant;
  children: ReactNode;
};

function Callout({ type = "note", children }: CalloutProps) {
  return (
    <div
      style={{
        borderLeft: `4px solid ${calloutBorderColors[type]}`,
        padding: "0.75rem 1rem",
        margin: "1rem 0",
      }}
    >
      {children}
    </div>
  );
}

export function useMDXComponents(components: MDXComponents): MDXComponents {
  return {
    ...components,
    pre: (props) => <CodeBlock {...props} />,
    Callout,
  };
}
