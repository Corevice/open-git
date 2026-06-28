"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const LOCALES = ["ja", "en"] as const;

type Locale = (typeof LOCALES)[number];

const VERSIONS = ["latest", "v1.0"];

function parseLocaleFromPath(pathname: string): {
  locale: Locale;
  versionPrefix: string;
  restOfPath: string;
} {
  const segments = pathname.split("/").filter(Boolean);
  let index = 0;

  let versionPrefix = "";
  if (segments[0] && VERSIONS.includes(segments[0])) {
    versionPrefix = `/${segments[0]}`;
    index = 1;
  }

  const maybeLocale = segments[index];
  if (maybeLocale === "ja" || maybeLocale === "en") {
    const rest = segments.slice(index + 1).join("/");
    return {
      locale: maybeLocale,
      versionPrefix,
      restOfPath: rest ? `/${rest}` : "",
    };
  }

  return { locale: "ja", versionPrefix, restOfPath: pathname === "/" ? "" : pathname };
}

export function LanguageSelector() {
  const pathname = usePathname();
  const { locale, versionPrefix, restOfPath } = parseLocaleFromPath(pathname);

  return (
    <nav aria-label="Language selector">
      {LOCALES.map((loc) => {
        const href = `${versionPrefix}/${loc}${restOfPath}`;
        const isActive = loc === locale;

        return (
          <Link
            key={loc}
            href={href}
            aria-current={isActive ? "page" : undefined}
            style={{ fontWeight: isActive ? "bold" : "normal" }}
          >
            {loc}
          </Link>
        );
      })}
    </nav>
  );
}
