"use client";

import { useCallback, useEffect, useRef } from "react";

const MAX_TILT_DEG = 6;
const TILT_LERP = 0.14;
const PERSPECTIVE_PX = 2200;

type TiltCardProps = {
  children: React.ReactNode;
  className?: string;
};

function clamp(value: number, min: number, max: number) {
  return Math.min(Math.max(value, min), max);
}

export function TiltCard({ children, className }: TiltCardProps) {
  const wrapperRef = useRef<HTMLDivElement>(null);
  const innerRef = useRef<HTMLDivElement>(null);
  const targetRef = useRef<[number, number]>([0, 0]);
  const currentRef = useRef<[number, number]>([0, 0]);
  const rafRef = useRef(0);
  const reducedMotionRef = useRef(false);

  const startLoopRef = useRef<() => void>(() => {});

  useEffect(() => {
    const tick = () => {
      const [tx, ty] = targetRef.current;
      const [cx, cy] = currentRef.current;
      const nx = cx + (tx - cx) * TILT_LERP;
      const ny = cy + (ty - cy) * TILT_LERP;
      currentRef.current = [nx, ny];

      const inner = innerRef.current;
      if (inner) {
        // Orthographic-leaning: large perspective + small angles → near-flat
        // depth, just enough to catch light. Y rotates with cursor X, X
        // rotates inversely with cursor Y to match the eye's expectation.
        const rotX = -ny * MAX_TILT_DEG;
        const rotY = nx * MAX_TILT_DEG;
        inner.style.transform = `rotateX(${rotX.toFixed(3)}deg) rotateY(${rotY.toFixed(3)}deg)`;
        const px = ((nx + 1) * 50).toFixed(2);
        const py = ((ny + 1) * 50).toFixed(2);
        inner.style.setProperty("--tilt-shine-x", `${px}%`);
        inner.style.setProperty("--tilt-shine-y", `${py}%`);
      }

      const settled =
        Math.abs(nx - tx) < 0.0008 && Math.abs(ny - ty) < 0.0008;
      if (settled) {
        rafRef.current = 0;
        return;
      }
      rafRef.current = requestAnimationFrame(tick);
    };
    const startLoop = () => {
      if (rafRef.current) return;
      rafRef.current = requestAnimationFrame(tick);
    };
    startLoopRef.current = startLoop;

    if (typeof window === "undefined" || typeof window.matchMedia !== "function")
      return;
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    reducedMotionRef.current = mq.matches;
    const onChange = (event: MediaQueryListEvent) => {
      reducedMotionRef.current = event.matches;
      if (event.matches) {
        targetRef.current = [0, 0];
        startLoop();
      }
    };
    mq.addEventListener("change", onChange);
    return () => {
      mq.removeEventListener("change", onChange);
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current);
        rafRef.current = 0;
      }
    };
  }, []);

  const handleMove = useCallback(
    (event: React.PointerEvent<HTMLDivElement>) => {
      if (reducedMotionRef.current) return;
      const wrap = wrapperRef.current;
      if (!wrap) return;
      const rect = wrap.getBoundingClientRect();
      if (rect.width === 0 || rect.height === 0) return;
      const x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
      const y = ((event.clientY - rect.top) / rect.height) * 2 - 1;
      targetRef.current = [clamp(x, -1, 1), clamp(y, -1, 1)];
      startLoopRef.current();
    },
    [],
  );

  const handleLeave = useCallback(() => {
    targetRef.current = [0, 0];
    startLoopRef.current();
  }, []);

  return (
    <div
      ref={wrapperRef}
      onPointerMove={handleMove}
      onPointerLeave={handleLeave}
      style={{ perspective: `${PERSPECTIVE_PX}px` }}
      className={className}
      data-testid="tilt-card"
    >
      <div
        ref={innerRef}
        className="relative h-full w-full overflow-hidden rounded-2xl will-change-transform [transform-style:preserve-3d] [transition:transform_220ms_cubic-bezier(0.2,0.7,0.2,1)]"
      >
        {children}
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 rounded-2xl opacity-70 mix-blend-screen"
          style={{
            background:
              "radial-gradient(circle at var(--tilt-shine-x, 50%) var(--tilt-shine-y, 50%), rgba(255,255,255,0.16) 0%, rgba(255,255,255,0.06) 28%, transparent 55%)",
          }}
        />
      </div>
    </div>
  );
}
