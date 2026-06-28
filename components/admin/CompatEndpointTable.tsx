"use client";

import { Fragment, useState } from "react";

import type { CompatEndpoint } from "@/app/admin/compatibility/types";

interface CompatEndpointTableProps {
  endpoints: CompatEndpoint[];
}

function methodBadgeClass(method: string): string {
  switch (method.toUpperCase()) {
    case "GET":
      return "bg-[#ddf4ff] text-[#0969da] border-[#54aeff]";
    case "POST":
      return "bg-[#dafbe1] text-[#1a7f37] border-[#4ac26b]";
    case "PUT":
    case "PATCH":
      return "bg-[#fff8c5] text-[#9a6700] border-[#d4a72c]";
    case "DELETE":
      return "bg-[#ffebe9] text-[#cf222e] border-[#ff8182]";
    default:
      return "bg-[#f6f8fa] text-[#656d76] border-[#d0d7de]";
  }
}

function statusChipClass(status: CompatEndpoint["status"]): string {
  switch (status) {
    case "pass":
      return "bg-[#dafbe1] text-[#1a7f37] border-[#4ac26b]";
    case "fail":
      return "bg-[#ffebe9] text-[#cf222e] border-[#ff8182]";
    case "unimplemented":
      return "bg-[#f6f8fa] text-[#656d76] border-[#d0d7de]";
  }
}

function formatLastRun(iso?: string): string {
  if (!iso) return "—";
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return date.toLocaleString(undefined, {
    dateStyle: "short",
    timeStyle: "short",
  });
}

function endpointKey(endpoint: CompatEndpoint): string {
  return `${endpoint.method}:${endpoint.path}`;
}

export function CompatEndpointTable({ endpoints }: CompatEndpointTableProps) {
  const [expandedKey, setExpandedKey] = useState<string | null>(null);

  const toggleRow = (endpoint: CompatEndpoint) => {
    if (endpoint.status !== "fail") return;
    const key = endpointKey(endpoint);
    setExpandedKey((current) => (current === key ? null : key));
  };

  return (
    <div className="overflow-hidden rounded-md border border-[#d0d7de] bg-white">
      <div className="border-b border-[#d0d7de] px-6 py-4">
        <h2 className="text-lg font-semibold text-[#24292f]">Endpoints</h2>
      </div>
      {endpoints.length === 0 ? (
        <p className="p-4 text-sm text-[#656d76]">No endpoint results.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full table-auto text-sm">
            <thead className="border-b border-[#d0d7de] bg-[#f6f8fa] text-left text-xs uppercase text-[#656d76]">
              <tr>
                <th className="px-4 py-2">Method</th>
                <th className="px-4 py-2">Path</th>
                <th className="px-4 py-2">Status</th>
                <th className="px-4 py-2">Last Run</th>
              </tr>
            </thead>
            <tbody>
              {endpoints.map((endpoint) => {
                const key = endpointKey(endpoint);
                const isExpanded = expandedKey === key;
                const isFail = endpoint.status === "fail";
                const diff = endpoint.diff ?? [];

                return (
                  <Fragment key={key}>
                    <tr
                      onClick={() => toggleRow(endpoint)}
                      className={`border-b border-[#eaeef2] last:border-b-0 ${
                        isFail
                          ? "cursor-pointer hover:bg-[#fff8f8]"
                          : "hover:bg-[#f6f8fa]"
                      }`}
                    >
                      <td className="px-4 py-2">
                        <span
                          className={`inline-flex rounded-md border px-2 py-0.5 font-mono text-xs font-semibold ${methodBadgeClass(endpoint.method)}`}
                        >
                          {endpoint.method.toUpperCase()}
                        </span>
                      </td>
                      <td className="px-4 py-2 font-mono text-xs text-[#24292f]">
                        {endpoint.path}
                      </td>
                      <td className="px-4 py-2">
                        <span
                          className={`inline-flex rounded-full border px-2.5 py-0.5 text-xs font-semibold capitalize ${statusChipClass(endpoint.status)}`}
                        >
                          {endpoint.status}
                        </span>
                      </td>
                      <td className="px-4 py-2 text-xs text-[#656d76]">
                        {formatLastRun(endpoint.last_run)}
                        {isFail && diff.length > 0 && (
                          <span className="ml-2 text-[#0969da]">
                            {isExpanded ? "▲" : "▼"}
                          </span>
                        )}
                      </td>
                    </tr>
                    {isFail && isExpanded && diff.length > 0 && (
                      <tr className="bg-[#fff8f8]">
                        <td colSpan={4} className="px-4 py-3">
                          <table className="w-full text-xs">
                            <thead>
                              <tr className="text-left text-[#656d76]">
                                <th className="pb-2 pr-4 font-medium">Field</th>
                                <th className="pb-2 pr-4 font-medium">
                                  Expected
                                </th>
                                <th className="pb-2 font-medium">Actual</th>
                              </tr>
                            </thead>
                            <tbody>
                              {diff.map((item) => (
                                <tr
                                  key={`${key}-${item.field}`}
                                  className="border-t border-[#ffebe9]"
                                >
                                  <td className="py-2 pr-4 font-mono text-[#24292f]">
                                    {item.field}
                                  </td>
                                  <td className="py-2 pr-4 font-mono text-[#1a7f37]">
                                    {item.expected}
                                  </td>
                                  <td className="py-2 font-mono text-[#cf222e]">
                                    {item.actual}
                                  </td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </td>
                      </tr>
                    )}
                  </Fragment>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
