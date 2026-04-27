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
  /**
   * Cursor acts as a gravity well. Each dot is pulled toward it; the
   * displacement falls off smoothly to zero at `gravityRadius`. Set to 0 to
   * disable warping. Defaults to ~2.5x dotSpacing so the deformation reads on
   * tight grids.
   */
  gravityStrength?: number;
  /** Influence radius of the gravity well. Defaults to 1.6x spotlightSize. */
  gravityRadius?: number;
  /**
   * Optional overlay rendered between the canvas and the children. Useful for
   * a vignette/scrim that keeps overlaid type readable without dimming the
   * children themselves.
   */
  scrim?: React.ReactNode;
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
  gravityStrength,
  gravityRadius,
  scrim,
}: LuminousGridProps) {
  const gravityPull = gravityStrength ?? dotSpacing * 2.5;
  const gravityReach = gravityRadius ?? spotlightSize * 1.6;
  const presenceRef = useRef(0);
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
    const wellRadius = mouse && gravityPull > 0 ? gravityReach : 0;

    // Presence: 1 while cursor is over the section, eases to 0 once it leaves.
    // The whole field's alpha is multiplied by this, so the mesh fades in on
    // enter and out on leave instead of being permanently visible.
    const presenceTarget = mouse !== null ? 1 : 0;
    presenceRef.current += (presenceTarget - presenceRef.current) * 0.08;
    if (Math.abs(presenceTarget - presenceRef.current) < 0.001) {
      presenceRef.current = presenceTarget;
    }
    const presence = presenceRef.current;
    if (presence < 0.005) return;

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
    // Traveling diagonal wave of brightness — gives the field a continuous
    // sense of flow even when the cursor isn't over it. ~10s for the band to
    // cross the hero; pow(...,6) sharpens the sine into a soft moving band
    // rather than continuous oscillation.
    const waveSpeed = 0.6;
    const waveFreq = 0.008;
    for (const { x, y, key } of dotPositionsRef.current) {
      // Gravity well: pull dot toward mouse with quadratic falloff to zero at
      // wellRadius. Capping displacement at the actual distance prevents dots
      // from overshooting the cursor.
      let drawX = x;
      let drawY = y;
      if (wellRadius > 0 && mouse) {
        const wdx = mouse.x - x;
        const wdy = mouse.y - y;
        const wdist = Math.sqrt(wdx * wdx + wdy * wdy);
        if (wdist > 0.001 && wdist < wellRadius) {
          const falloff = 1 - wdist / wellRadius;
          const pull = Math.min(wdist, gravityPull * falloff * falloff);
          drawX = x + (wdx / wdist) * pull;
          drawY = y + (wdy / wdist) * pull;
        }
      }

      const off = blinkOffsetsRef.current.get(key) ?? 0;
      const spd = blinkSpeedsRef.current.get(key) ?? 1;
      const inten = blinkIntensitiesRef.current.get(key) ?? 1;
      const blink = Math.sin(t * spd + off) * 0.5 + 0.5;
      const alpha = enableBlinking ? 0.32 + blink * inten * 0.68 : 1;

      const wavePhase = (x + y * 0.6) * waveFreq - t * waveSpeed;
      const waveSine = 0.5 + 0.5 * Math.sin(wavePhase);
      const wave = waveSine * waveSine * waveSine * waveSine * waveSine * waveSine;

      let spotlight = 0;
      if (spotlightRadius > 0 && mouse) {
        const dx = drawX - mouse.x;
        const dy = drawY - mouse.y;
        const dist = Math.sqrt(dx * dx + dy * dy);
        spotlight = Math.max(0, 1 - dist / spotlightRadius);
      }

      const light = Math.max(spotlight, wave * 0.75);
      const litAlpha = Math.min(1, alpha + wave * 0.55);
      const size =
        light > 0 ? dotSize + light * dotSize * 1.5 : dotSize;
      if (light > 0.04) {
        const a = (0.7 + light * 0.3) * litAlpha * presence;
        ctx.fillStyle = `rgba(${colors.highlightR}, ${colors.highlightG}, ${colors.highlightB}, ${a})`;
      } else {
        ctx.fillStyle = `rgba(${colors.baseR}, ${colors.baseG}, ${colors.baseB}, ${litAlpha * presence})`;
      }
      ctx.beginPath();
      ctx.arc(drawX, drawY, size, 0, Math.PI * 2);
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
    gravityPull,
    gravityReach,
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
      currentMouseRef.current !== null || presenceRef.current > 0.005;
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
            (currentMouseRef.current !== null || presenceRef.current > 0.005)
          ) {
            rafRef.current = requestAnimationFrame(loop);
          } else {
            isAnimatingRef.current = false;
          }
        };
        rafRef.current = requestAnimationFrame(loop);
      }
    },
    [draw, hoverHighlight],
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
      {scrim}
      <div className="relative">{children}</div>
    </section>
  );
}
