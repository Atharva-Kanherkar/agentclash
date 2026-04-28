"use client";

import { useEffect, useState, type ComponentType } from "react";

const FRAMER_URL =
  "https://framer.com/m/SpectraNoise-l2Js.js@PswA93ZW5ISUVLr5J1Ed";

// Use the Function constructor so neither TS nor the bundler tries to resolve
// the URL at build time — the import happens natively in the browser.
const importUrl = new Function("u", "return import(u)") as (
  u: string,
) => Promise<{ default?: ComponentType<unknown> } & Record<string, unknown>>;

type SpectraNoiseProps = {
  className?: string;
  style?: React.CSSProperties;
} & Record<string, unknown>;

export function SpectraNoise({ style, ...props }: SpectraNoiseProps) {
  const [Comp, setComp] = useState<ComponentType<SpectraNoiseProps> | null>(
    null,
  );

  useEffect(() => {
    let cancelled = false;
    importUrl(FRAMER_URL)
      .then((mod) => {
        if (cancelled) return;
        const Resolved = (mod.default ?? mod) as ComponentType<SpectraNoiseProps>;
        setComp(() => Resolved);
      })
      .catch(() => {
        // Swallow — background is decorative; section degrades to plain bg.
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const fillStyle: React.CSSProperties = {
    width: "100%",
    height: "100%",
    ...style,
  };

  if (!Comp) return <div aria-hidden style={fillStyle} />;
  return <Comp {...props} style={fillStyle} />;
}
