"use client";

import Prism from "prismjs";
import "prismjs/components/prism-typescript";
import { useEffect, useMemo, useRef, type ReactElement, type ReactNode } from "react";
import CopyButton from "./CopyButton";

type CodeBlockProps = {
  children?: ReactNode;
  className?: string;
};

function extractCode(children: ReactNode): string {
  if (typeof children === "string") {
    return children.replace(/\n$/, "");
  }

  if (Array.isArray(children)) {
    return children.map((child) => extractCode(child)).join("");
  }

  if (children && typeof children === "object" && "props" in children) {
    return extractCode((children as ReactElement<{ children?: ReactNode }>).props.children);
  }

  return "";
}

export default function CodeBlock({ children, className }: CodeBlockProps) {
  const codeRef = useRef<HTMLElement>(null);
  const code = useMemo(() => extractCode(children), [children]);

  useEffect(() => {
    if (codeRef.current) {
      Prism.highlightElement(codeRef.current);
    }
  }, [code, className]);

  return (
    <div style={{ position: "relative" }}>
      <CopyButton code={code} />
      <pre>
        <code ref={codeRef} className={className ?? undefined}>
          {code}
        </code>
      </pre>
    </div>
  );
}
