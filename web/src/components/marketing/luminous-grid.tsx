"use client";

/*
 * LuminousGrid — animated dot grid with cursor-follow spotlight.
 * Vendored from Framer's free LuminousGrid asset
 * (https://framer.com/m/LuminousGrid-HAie.js), stripped of Framer-only
 * imports (addPropertyControls, useIsStaticRenderer) and exposed as a plain
 * React section wrapper. Mouse state lives in refs; the only state-driven
 * re-render is for the spotlight enable/disable signal.
 */

import { useCallback, useEffect, useMemo, useRef } from "react";

export type LuminousGridProps = {
  children?: React.ReactNode;
  className?: string;
  dotSize?: number;
  dotSpacing?: number;
  dotColor?: string;
  spotlightSize?: number;
  hoverHighlight?: boolean;
  enableBlinking?: boolean;
};

type RGB = { r: number; g: number; b: number };

function parseColor(color: string): RGB {
  if (typeof document === "undefined") return { r: 204, g: 204, b: 204 };
  const c = document.createElement("canvas");
  c.width = 1;
  c.height = 1;
  const ctx = c.getContext("2d");
  if (!ctx) return { r: 204, g: 204, b: 204 };
  ctx.fillStyle = color;
  ctx.fillRect(0, 0, 1, 1);
  const d = ctx.getImageData(0, 0, 1, 1).data;
  return { r: d[0], g: d[1], b: d[2] };
}

export function LuminousGrid({
  children,
  className,
  dotSize = 2,
  dotSpacing = 30,
  dotColor = "#CCCCCC",
  spotlightSize = 150,
  hoverHighlight = true,
  enableBlinking = true,
}: LuminousGridProps) {
  const sectionRef = useRef<HTMLElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const rafRef = useRef(0);
  const isAnimatingRef = useRef(false);
  const currentMouseRef = useRef<{ x: number; y: number } | null>(null);
  const blinkOffsetsRef = useRef(new Map<string, number>());
  const blinkSpeedsRef = useRef(new Map<string, number>());
  const blinkIntensitiesRef = useRef(new Map<string, number>());
  const startTimeRef = useRef(0);
  const lastMouseUpdateRef = useRef(0);
  const dotPositionsRef = useRef<{ x: number; y: number; key: string }[]>([]);

  const colors = useMemo(() => {
    const { r, g, b } = parseColor(dotColor);
    const lighten = 0.6;
    return {
      baseR: r,
      baseG: g,
      baseB: b,
      highlightR: Math.min(255, r + (255 - r) * lighten),
      highlightG: Math.min(255, g + (255 - g) * lighten),
      highlightB: Math.min(255, b + (255 - b) * lighten),
    };
  }, [dotColor]);

  const draw = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d", { alpha: true });
    if (!ctx) return;
    const rect = canvas.getBoundingClientRect();
    const w = rect.width;
    const h = rect.height;
    ctx.clearRect(0, 0, w, h);

    const mouse = currentMouseRef.current;
    const spotlightRadius = hoverHighlight && mouse ? spotlightSize * 0.8 : 0;

    if (dotPositionsRef.current.length === 0) {
      const cols = Math.ceil(w / dotSpacing) + 1;
      const rows = Math.ceil(h / dotSpacing) + 1;
      const positions: { x: number; y: number; key: string }[] = [];
      for (let i = 0; i < cols; i++) {
        for (let j = 0; j < rows; j++) {
          const key = `${i}-${j}`;
          positions.push({ x: i * dotSpacing, y: j * dotSpacing, key });
          if (!blinkOffsetsRef.current.has(key)) {
            blinkOffsetsRef.current.set(key, Math.random() * Math.PI * 2);
            blinkSpeedsRef.current.set(key, 0.5 + Math.random() * 1.5);
            blinkIntensitiesRef.current.set(key, 0.5 + Math.random() * 0.5);
          }
        }
      }
      dotPositionsRef.current = positions;
    }

    const t = (Date.now() - startTimeRef.current) / 1000;
    for (const { x, y, key } of dotPositionsRef.current) {
      const off = blinkOffsetsRef.current.get(key) ?? 0;
      const spd = blinkSpeedsRef.current.get(key) ?? 1;
      const inten = blinkIntensitiesRef.current.get(key) ?? 1;
      const blink = Math.sin(t * spd + off) * 0.5 + 0.5;
      const alpha = enableBlinking ? 0.2 + blink * inten * 0.8 : 1;

      let intensity = 0;
      if (spotlightRadius > 0 && mouse) {
        const dx = x - mouse.x;
        const dy = y - mouse.y;
        const dist = Math.sqrt(dx * dx + dy * dy);
        intensity = Math.max(0, 1 - dist / spotlightRadius);
      }
      const size =
        intensity > 0 ? dotSize + intensity * dotSize * 1.5 : dotSize;
      if (intensity > 0) {
        const a = (0.7 + intensity * 0.3) * alpha;
        ctx.fillStyle = `rgba(${colors.highlightR}, ${colors.highlightG}, ${colors.highlightB}, ${a})`;
      } else {
        ctx.fillStyle = `rgba(${colors.baseR}, ${colors.baseG}, ${colors.baseB}, ${alpha})`;
      }
      ctx.beginPath();
      ctx.arc(x, y, size, 0, Math.PI * 2);
      ctx.fill();
    }
  }, [
    colors.baseR,
    colors.baseG,
    colors.baseB,
    colors.highlightR,
    colors.highlightG,
    colors.highlightB,
    dotSize,
    dotSpacing,
    enableBlinking,
    hoverHighlight,
    spotlightSize,
  ]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d", { alpha: true });
    if (!ctx) return;
    if (startTimeRef.current === 0) startTimeRef.current = Date.now();
    const dpr =
      typeof window !== "undefined" ? window.devicePixelRatio || 1 : 1;
    const resize = () => {
      const rect = canvas.getBoundingClientRect();
      canvas.width = rect.width * dpr;
      canvas.height = rect.height * dpr;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      dotPositionsRef.current = [];
    };
    resize();

    const ro =
      typeof ResizeObserver === "undefined" ? null : new ResizeObserver(resize);
    ro?.observe(canvas);

    const shouldAnimate = () =>
      enableBlinking || currentMouseRef.current !== null;
    isAnimatingRef.current = true;
    const loop = () => {
      draw();
      if (isAnimatingRef.current && shouldAnimate()) {
        rafRef.current = requestAnimationFrame(loop);
      } else {
        isAnimatingRef.current = false;
      }
    };
    if (shouldAnimate()) {
      rafRef.current = requestAnimationFrame(loop);
    } else {
      draw();
    }

    return () => {
      isAnimatingRef.current = false;
      if (rafRef.current) cancelAnimationFrame(rafRef.current);
      ro?.disconnect();
    };
  }, [draw, enableBlinking]);

  const handleMove = useCallback(
    (event: React.PointerEvent<HTMLElement>) => {
      if (!hoverHighlight) return;
      const now = Date.now();
      if (now - lastMouseUpdateRef.current < 16) return;
      lastMouseUpdateRef.current = now;
      const canvas = canvasRef.current;
      if (!canvas) return;
      const rect = canvas.getBoundingClientRect();
      currentMouseRef.current = {
        x: event.clientX - rect.left,
        y: event.clientY - rect.top,
      };
      if (!isAnimatingRef.current) {
        isAnimatingRef.current = true;
        const loop = () => {
          draw();
          if (
            isAnimatingRef.current &&
            (enableBlinking || currentMouseRef.current !== null)
          ) {
            rafRef.current = requestAnimationFrame(loop);
          } else {
            isAnimatingRef.current = false;
          }
        };
        rafRef.current = requestAnimationFrame(loop);
      }
    },
    [draw, enableBlinking, hoverHighlight],
  );

  const handleLeave = useCallback(() => {
    currentMouseRef.current = null;
  }, []);

  return (
    <section
      ref={sectionRef}
      onPointerMove={handleMove}
      onPointerLeave={handleLeave}
      className={`relative ${className ?? ""}`}
      data-testid="luminous-grid"
    >
      <canvas
        ref={canvasRef}
        aria-hidden
        className="pointer-events-none absolute inset-0 block size-full"
      />
      <div className="relative">{children}</div>
    </section>
  );
}
