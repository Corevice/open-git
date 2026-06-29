"use client";

import { useEffect, useState } from "react";

import { getAppLicenses } from "@/lib/api";
import type { AppLicenses } from "@/lib/api-types";

export default function LicensesPage() {
  const [licenses, setLicenses] = useState<AppLicenses | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getAppLicenses(process.env.NEXT_PUBLIC_API_BASE_URL ?? "")
      .then(setLicenses)
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <div className="mx-auto max-w-4xl px-6 py-12">
        <p className="text-muted-foreground">Loading...</p>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl px-6 py-12">
      <h1 className="mb-6 text-3xl font-bold">Licenses</h1>

      <section className="mb-8">
        <h2 className="mb-2 text-xl font-semibold">Project License</h2>
        <p>{licenses?.app_license ?? ""}</p>
      </section>

      {licenses?.third_party.length === 0 ? (
        <p>ライセンス情報を取得できませんでした</p>
      ) : (
        <section>
          <h2 className="mb-4 text-xl font-semibold">Third-Party Licenses</h2>
          <table className="w-full border-collapse text-left">
            <thead>
              <tr className="border-b">
                <th className="py-2 pr-4">Name</th>
                <th className="py-2 pr-4">Version</th>
                <th className="py-2 pr-4">License</th>
                <th className="py-2">URL</th>
              </tr>
            </thead>
            <tbody>
              {licenses?.third_party.map((entry) => (
                <tr key={`${entry.name}-${entry.version}`} className="border-b">
                  <td className="py-2 pr-4">{entry.name}</td>
                  <td className="py-2 pr-4">{entry.version}</td>
                  <td className="py-2 pr-4">{entry.license}</td>
                  <td className="py-2">
                    <a
                      href={entry.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-primary underline"
                    >
                      {entry.url}
                    </a>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      )}
    </div>
  );
}
