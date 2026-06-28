"use client";

import { usePathname, useRouter } from "next/navigation";

const VERSIONS = ["latest", "v1.0"] as const;

type Version = (typeof VERSIONS)[number];

function parseVersionFromPath(pathname: string): {
  version: Version;
  restOfPath: string;
} {
  const segments = pathname.split("/").filter(Boolean);
  const first = segments[0];

  if (first && VERSIONS.includes(first as Version)) {
    const rest = segments.slice(1).join("/");
    return { version: first as Version, restOfPath: rest ? `/${rest}` : "" };
  }

  return { version: "latest", restOfPath: pathname === "/" ? "" : pathname };
}

export function VersionSelector() {
  const pathname = usePathname();
  const router = useRouter();
  const { version, restOfPath } = parseVersionFromPath(pathname);

  const handleChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    const newVersion = event.target.value;
    router.push(`/${newVersion}${restOfPath}`);
  };

  return (
    <select
      aria-label="Documentation version"
      value={version}
      onChange={handleChange}
    >
      {VERSIONS.map((v) => (
        <option key={v} value={v}>
          {v}
        </option>
      ))}
    </select>
  );
}
