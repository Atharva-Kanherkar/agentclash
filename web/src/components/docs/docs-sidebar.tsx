"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Search } from "lucide-react";
import type { DocNavSection, DocSearchItem } from "@/lib/docs";

export function DocsSidebar({
  sections,
  currentHref,
}: {
  sections: DocNavSection[];
  currentHref: string;
}) {
  const [query, setQuery] = useState("");
  const [searchItems, setSearchItems] = useState<DocSearchItem[]>([]);
  const [searchState, setSearchState] = useState<
    "idle" | "loading" | "ready" | "error"
  >("idle");
  const normalized = query.trim().toLowerCase();
  const tokens = normalized.split(/\s+/).filter(Boolean);
  const matches =
    tokens.length === 0
      ? []
      : searchItems
          .filter((item) =>
            tokens.every((token) => item.searchText.includes(token)),
          )
          .slice(0, 12);

  useEffect(() => {
    if (searchState !== "loading") return;

    let cancelled = false;

    fetch("/docs/search.json", {
      headers: {
        Accept: "application/json",
      },
    })
      .then((response) => {
        if (!response.ok) {
          throw new Error(`Failed to load docs search index: ${response.status}`);
        }
        return response.json() as Promise<DocSearchItem[]>;
      })
      .then((items) => {
        if (cancelled) return;
        setSearchItems(items);
        setSearchState("ready");
      })
      .catch(() => {
        if (cancelled) return;
        setSearchState("error");
      });

    return () => {
      cancelled = true;
    };
  }, [searchState]);

  const ensureSearchLoaded = () => {
    if (searchState === "idle" || searchState === "error") {
      setSearchState("loading");
    }
  };

  return (
    <div className="rounded-[28px] border border-white/[0.08] bg-white/[0.03] p-4 sm:p-5">
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-white/30" />
        <input
          value={query}
          onChange={(event) => {
            ensureSearchLoaded();
            setQuery(event.target.value);
          }}
          onFocus={ensureSearchLoaded}
          placeholder="Search docs"
          className="h-11 w-full rounded-2xl border border-white/[0.08] bg-black/20 pl-10 pr-4 text-sm text-white outline-none transition-colors placeholder:text-white/28 focus:border-lime-200/30"
        />
      </div>

      {tokens.length > 0 ? (
        <div className="mt-4">
          <p className="px-1 text-[11px] uppercase tracking-[0.18em] text-white/30">
            Search Results
          </p>
          <div className="mt-2 space-y-1">
            {matches.map((item) => {
              const active = currentHref === item.href;
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`block rounded-2xl px-3 py-2.5 transition-colors ${
                    active
                      ? "bg-lime-300/[0.12] text-lime-100"
                      : "text-white/68 hover:bg-white/[0.04] hover:text-white"
                  }`}
                >
                  <span className="block text-sm font-medium">{item.title}</span>
                  <span
                    className={`mt-1 block text-xs leading-5 ${
                      active ? "text-lime-100/70" : "text-white/42"
                    }`}
                  >
                    {item.description}
                  </span>
                </Link>
              );
            })}
            {searchState === "loading" && (
              <p className="px-3 py-3 text-sm text-white/40">
                Loading docs index...
              </p>
            )}
            {searchState === "error" && (
              <p className="px-3 py-3 text-sm text-white/40">
                Search is temporarily unavailable.
              </p>
            )}
            {matches.length === 0 && (
              <p className="px-3 py-3 text-sm text-white/40">
                {searchState === "ready"
                  ? "No docs matched that query."
                  : "Keep typing to search the docs."}
              </p>
            )}
          </div>
        </div>
      ) : (
        <div className="mt-5 space-y-5">
          <Link
            href="/docs"
            className={`block rounded-2xl px-3 py-2.5 transition-colors ${
              currentHref === "/docs"
                ? "bg-lime-300/[0.12] text-lime-100"
                : "text-white/70 hover:bg-white/[0.04] hover:text-white"
            }`}
          >
            <span className="block text-sm font-medium">Overview</span>
            <span
              className={`mt-1 block text-xs leading-5 ${
                currentHref === "/docs" ? "text-lime-100/70" : "text-white/45"
              }`}
            >
              Start here if you want the shortest path to understanding the product.
            </span>
          </Link>

          {sections.map((section) => (
            <div key={section.title}>
              <p className="px-3 text-[11px] uppercase tracking-[0.18em] text-white/30">
                {section.title}
              </p>
              <div className="mt-2 space-y-1">
                {section.items.map((item) => {
                  const active = currentHref === item.href;

                  return (
                    <Link
                      key={item.href}
                      href={item.href}
                      className={`block rounded-2xl px-3 py-2.5 transition-colors ${
                        active
                          ? "bg-lime-300/[0.12] text-lime-100"
                          : "text-white/68 hover:bg-white/[0.04] hover:text-white"
                      }`}
                    >
                      <span className="block text-sm font-medium">
                        {item.title}
                      </span>
                      <span
                        className={`mt-1 block text-xs leading-5 ${
                          active ? "text-lime-100/70" : "text-white/42"
                        }`}
                      >
                        {item.description}
                      </span>
                    </Link>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
