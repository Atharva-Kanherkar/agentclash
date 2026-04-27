"use client";

/*
 * Starfield — r3f / WebGL hyperspace background.
 *
 * Behaviors (mirroring the prior lightspeed component):
 *   • cursor parallax: the field tilts toward the pointer
 *   • hover speed boost: forward velocity ramps up while the pointer is over
 *     the section, eases back to baseline on leave
 *   • device gyro: mobile orientation drives the same tilt target as the
 *     cursor (no iOS permission chip; pointer fallback covers iOS until the
 *     user grants motion access elsewhere)
 *
 * Visuals: vivid colored stars at varied sizes, additive-blended on black.
 * A custom ShaderMaterial supports per-vertex size so a small fraction of
 * stars render as bright "hero" blooms among many pinpricks.
 */

import { Canvas, useFrame } from "@react-three/fiber";
import { useEffect, useMemo, useRef } from "react";
import * as THREE from "three";

export type StarfieldProps = {
  /** Number of stars. */
  count?: number;
  /** Palette to randomly tint stars from. */
  colors?: string[];
  /** Baseline forward drift in world units / second. */
  velocity?: number;
  /** How far the field tilts in response to cursor / gyro (radians). */
  parallax?: number;
  /** Multiplier applied to velocity while the pointer is over the section. */
  hoverBoost?: number;
  className?: string;
};

const DEFAULT_COLORS = [
  "#ff3b3b", // red
  "#ff63b8", // magenta/pink
  "#ffd96b", // gold
  "#ff9c3b", // orange
  "#3bd3ff", // cyan
  "#ffffff", // sparse white
];

const DEPTH = 80;
const HALF_DEPTH = DEPTH / 2;

const TILT_LERP = 0.08;
const SPEED_LERP = 0.06;
const GYRO_RANGE_DEG = 22;

// Deterministic PRNG so the arrangement is stable across renders and
// `react-hooks/purity` doesn't flag a Math.random call inside useMemo.
function mulberry32(seed: number) {
  let s = seed >>> 0;
  return () => {
    s = (s + 0x6d2b79f5) >>> 0;
    let t = s;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

function clamp(value: number, min: number, max: number) {
  return Math.min(Math.max(value, min), max);
}

const VERTEX_SHADER = /* glsl */ `
  attribute float aSize;
  attribute vec3 aColor;
  varying vec3 vColor;
  void main() {
    vColor = aColor;
    vec4 mvPosition = modelViewMatrix * vec4(position, 1.0);
    gl_PointSize = aSize * (300.0 / -mvPosition.z);
    gl_Position = projectionMatrix * mvPosition;
  }
`;

const FRAGMENT_SHADER = /* glsl */ `
  uniform sampler2D pointTexture;
  varying vec3 vColor;
  void main() {
    vec4 tex = texture2D(pointTexture, gl_PointCoord);
    if (tex.a < 0.01) discard;
    gl_FragColor = vec4(vColor, 1.0) * tex;
  }
`;

function buildStarTexture(): THREE.Texture {
  const size = 64;
  const canvas = document.createElement("canvas");
  canvas.width = size;
  canvas.height = size;
  const ctx = canvas.getContext("2d");
  if (ctx) {
    const r = size / 2;
    const gradient = ctx.createRadialGradient(r, r, 0, r, r, r);
    gradient.addColorStop(0, "rgba(255,255,255,1)");
    gradient.addColorStop(0.12, "rgba(255,255,255,0.95)");
    gradient.addColorStop(0.4, "rgba(255,255,255,0.35)");
    gradient.addColorStop(1, "rgba(255,255,255,0)");
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, size, size);
  }
  const texture = new THREE.CanvasTexture(canvas);
  texture.minFilter = THREE.LinearFilter;
  texture.magFilter = THREE.LinearFilter;
  return texture;
}

type StarsProps = {
  count: number;
  colors: string[];
  baseVelocity: number;
  parallax: number;
  hoverBoost: number;
  hoveringRef: React.RefObject<boolean>;
  tiltTargetRef: React.RefObject<{ x: number; y: number }>;
};

function Stars({
  count,
  colors,
  baseVelocity,
  parallax,
  hoverBoost,
  hoveringRef,
  tiltTargetRef,
}: StarsProps) {
  const pointsRef = useRef<THREE.Points>(null);
  const speedRef = useRef(1);

  const { positions, colorAttr, sizes, material } = useMemo(() => {
    const positions = new Float32Array(count * 3);
    const colorAttr = new Float32Array(count * 3);
    const sizes = new Float32Array(count);
    const palette = colors.map((c) => new THREE.Color(c));
    const rand = mulberry32(count * 9301 + 49297);

    for (let i = 0; i < count; i++) {
      positions[i * 3 + 0] = (rand() - 0.5) * DEPTH;
      positions[i * 3 + 1] = (rand() - 0.5) * DEPTH;
      positions[i * 3 + 2] = (rand() - 0.5) * DEPTH;

      const base = palette[Math.floor(rand() * palette.length)];
      const brightness = 0.45 + rand() * 0.55;
      colorAttr[i * 3 + 0] = base.r * brightness;
      colorAttr[i * 3 + 1] = base.g * brightness;
      colorAttr[i * 3 + 2] = base.b * brightness;

      // ~4% of stars are "hero" blooms; the rest are small pinpricks.
      sizes[i] =
        rand() < 0.04 ? 4.5 + rand() * 4 : 0.6 + rand() * 1.6;
    }

    const material = new THREE.ShaderMaterial({
      uniforms: { pointTexture: { value: buildStarTexture() } },
      vertexShader: VERTEX_SHADER,
      fragmentShader: FRAGMENT_SHADER,
      transparent: true,
      depthWrite: false,
      depthTest: false,
      blending: THREE.AdditiveBlending,
    });

    return { positions, colorAttr, sizes, material };
  }, [count, colors]);

  useEffect(() => {
    return () => {
      material.dispose();
      const tex = (
        material.uniforms.pointTexture as { value: THREE.Texture | null }
      ).value;
      tex?.dispose();
    };
  }, [material]);

  useFrame((_, delta) => {
    const points = pointsRef.current;
    if (!points) return;

    const speedTarget = hoveringRef.current ? hoverBoost : 1;
    speedRef.current += (speedTarget - speedRef.current) * SPEED_LERP;

    const arr = points.geometry.attributes.position.array as Float32Array;
    const step = baseVelocity * speedRef.current * delta;
    for (let i = 2; i < arr.length; i += 3) {
      arr[i] += step;
      if (arr[i] > HALF_DEPTH) arr[i] -= DEPTH;
    }
    points.geometry.attributes.position.needsUpdate = true;

    const target = tiltTargetRef.current;
    const targetY = target.x * parallax;
    const targetX = -target.y * parallax;
    points.rotation.y += (targetY - points.rotation.y) * TILT_LERP;
    points.rotation.x += (targetX - points.rotation.x) * TILT_LERP;
  });

  return (
    <points ref={pointsRef}>
      <bufferGeometry>
        <bufferAttribute
          attach="attributes-position"
          args={[positions, 3]}
          count={count}
        />
        <bufferAttribute
          attach="attributes-aColor"
          args={[colorAttr, 3]}
          count={count}
        />
        <bufferAttribute
          attach="attributes-aSize"
          args={[sizes, 1]}
          count={count}
        />
      </bufferGeometry>
      <primitive object={material} attach="material" />
    </points>
  );
}

export function Starfield({
  count = 4500,
  colors = DEFAULT_COLORS,
  velocity = 7,
  parallax = 0.18,
  hoverBoost = 3.2,
  className,
}: StarfieldProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const hoveringRef = useRef(false);
  const tiltTargetRef = useRef({ x: 0, y: 0 });

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const onPointerMove = (event: PointerEvent) => {
      const rect = container.getBoundingClientRect();
      if (rect.width === 0 || rect.height === 0) return;
      const x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
      const y = ((event.clientY - rect.top) / rect.height) * 2 - 1;
      tiltTargetRef.current = {
        x: clamp(x, -1, 1),
        y: -clamp(y, -1, 1),
      };
    };
    const onPointerEnter = () => {
      hoveringRef.current = true;
    };
    const onPointerLeave = () => {
      hoveringRef.current = false;
      tiltTargetRef.current = { x: 0, y: 0 };
    };

    container.addEventListener("pointermove", onPointerMove, { passive: true });
    container.addEventListener("pointerenter", onPointerEnter, {
      passive: true,
    });
    container.addEventListener("pointerleave", onPointerLeave, {
      passive: true,
    });

    let orientationAttached = false;
    const onOrientation = (event: DeviceOrientationEvent) => {
      if (event.gamma == null || event.beta == null) return;
      tiltTargetRef.current = {
        x: clamp(event.gamma / GYRO_RANGE_DEG, -1, 1),
        y: -clamp(event.beta / GYRO_RANGE_DEG, -1, 1),
      };
    };
    if (typeof window !== "undefined" && "DeviceOrientationEvent" in window) {
      const ctor = window.DeviceOrientationEvent as typeof DeviceOrientationEvent & {
        requestPermission?: () => Promise<"granted" | "denied">;
      };
      // iOS gates DeviceOrientation behind a user-gesture permission prompt;
      // pointer parallax covers that case until the user grants it elsewhere.
      if (typeof ctor.requestPermission !== "function") {
        window.addEventListener("deviceorientation", onOrientation);
        orientationAttached = true;
      }
    }

    return () => {
      container.removeEventListener("pointermove", onPointerMove);
      container.removeEventListener("pointerenter", onPointerEnter);
      container.removeEventListener("pointerleave", onPointerLeave);
      if (orientationAttached) {
        window.removeEventListener("deviceorientation", onOrientation);
      }
    };
  }, []);

  return (
    <div
      ref={containerRef}
      className={className}
      style={{
        position: "absolute",
        inset: 0,
        background: "#000",
      }}
    >
      <Canvas
        camera={{ position: [0, 0, 30], fov: 60 }}
        dpr={[1, 1.5]}
        gl={{
          alpha: false,
          antialias: false,
          powerPreference: "high-performance",
        }}
        style={{ pointerEvents: "none" }}
      >
        <Stars
          count={count}
          colors={colors}
          baseVelocity={velocity}
          parallax={parallax}
          hoverBoost={hoverBoost}
          hoveringRef={hoveringRef}
          tiltTargetRef={tiltTargetRef}
        />
      </Canvas>
    </div>
  );
}
