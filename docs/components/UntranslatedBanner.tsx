"use client";

import { useState } from "react";

type UntranslatedBannerProps = {
  locale?: string;
  pageLang?: string;
};

export function UntranslatedBanner({
  locale,
  pageLang,
}: UntranslatedBannerProps) {
  const [dismissed, setDismissed] = useState(false);

  if (dismissed || locale !== "en" || pageLang !== "ja") {
    return null;
  }

  return (
    <div
      role="status"
      style={{
        backgroundColor: "#fef9c3",
        border: "1px solid #fde047",
        borderRadius: "6px",
        color: "#713f12",
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        gap: "12px",
        marginBottom: "16px",
        padding: "12px 16px",
      }}
    >
      <span>このページはまだ日本語のみ提供されています。</span>
      <button
        type="button"
        aria-label="閉じる"
        onClick={() => setDismissed(true)}
        style={{
          background: "transparent",
          border: "none",
          color: "#713f12",
          cursor: "pointer",
          fontSize: "18px",
          lineHeight: 1,
          padding: "0 4px",
        }}
      >
        ×
      </button>
    </div>
  );
}
