"use client";

import { useCallback, useEffect, useState } from "react";

type PagefindResult = {
  id: string;
  data: () => Promise<{ url: string; meta: { title: string } }>;
};

type PagefindInstance = {
  init?: () => Promise<void>;
  search: (query: string) => Promise<{ results: PagefindResult[] }>;
};

declare global {
  interface Window {
    pagefind?: PagefindInstance;
  }
}

type SearchResult = {
  url: string;
  title: string;
};

export default function Search() {
  const [available, setAvailable] = useState(false);
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);

  useEffect(() => {
    let cancelled = false;

    const loadPagefind = async () => {
      try {
        const pagefind = (await import(
          /* webpackIgnore: true */
          "/pagefind/pagefind.js"
        )) as PagefindInstance;

        if (pagefind.init) {
          await pagefind.init();
        }

        if (!cancelled) {
          window.pagefind = pagefind;
          setAvailable(true);
        }
      } catch {
        if (!cancelled) {
          setAvailable(false);
        }
      }
    };

    void loadPagefind();

    return () => {
      cancelled = true;
    };
  }, []);

  const handleSearch = useCallback(
    async (value: string) => {
      setQuery(value);

      if (!available || !window.pagefind || !value.trim()) {
        setResults([]);
        return;
      }

      const response = await window.pagefind.search(value);
      const items = await Promise.all(
        response.results.slice(0, 10).map(async (result) => {
          const data = await result.data();
          return { url: data.url, title: data.meta.title };
        }),
      );
      setResults(items);
    },
    [available],
  );

  return (
    <div>
      <input
        type="search"
        value={query}
        onChange={(event) => void handleSearch(event.target.value)}
        disabled={!available}
        aria-disabled={!available}
        placeholder={available ? "ドキュメントを検索" : "検索は利用できません"}
        aria-label="ドキュメント検索"
      />
      {available && results.length > 0 && (
        <ul role="listbox" aria-label="検索結果">
          {results.map((result) => (
            <li key={result.url} role="option">
              <a href={result.url}>{result.title}</a>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
