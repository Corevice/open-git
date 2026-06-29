"use client";

import { useCallback, useEffect, useRef, useState } from "react";

type CopyButtonProps = {
  code: string;
};

export default function CopyButton({ code }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  const handleCopy = useCallback(async () => {
    await navigator.clipboard.writeText(code);
    setCopied(true);

    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }

    timeoutRef.current = setTimeout(() => {
      setCopied(false);
    }, 2000);
  }, [code]);

  return (
    <button
      type="button"
      onClick={() => void handleCopy()}
      aria-label={copied ? "コードをコピーしました" : "コードをコピー"}
    >
      {copied ? "コピーしました！" : "コピー"}
    </button>
  );
}
