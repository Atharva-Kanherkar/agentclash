"use client";

/*
 * Starfield — r3f / WebGL background of drifting stars with mouse parallax.
 * Stars are distributed in a depth box, drift toward the camera, and wrap.
 * The whole field tilts subtly toward the cursor for a parallax feel.
 */

import { Canvas, useFrame } from "@react-three/fiber";
import { useEffect, useMemo, useRef } from "react";
import * as THREE from "three";

export type StarfieldProps = {
  /** Number of stars. Default 1500 — cheap on the GPU, dense to the eye. */
  count?: number;
  /** Palette to randomly tint stars from. Mostly white with cool/warm accents. */
  colors?: string[];
  /** Forward drift in world units / second. */
  velocity?: number;
  /** How far the field tilts in response to the cursor (radians). */
  parallax?: number;
  className?: string;
};

const DEFAULT_COLORS = ["#ffffff", "#cfe2ff", "#7eb8e6", "#f3d9b1"];
const DEPTH = 80;
const HALF_DEPTH = DEPTH / 2;

// Deterministic PRNG so the starfield arrangement is stable across renders
// (and `react-hooks/purity` doesn't flag a Math.random call inside useMemo).
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
    gradient.addColorStop(0.18, "rgba(255,255,255,0.85)");
    gradient.addColorStop(0.45, "rgba(255,255,255,0.25)");
    gradient.addColorStop(1, "rgba(255,255,255,0)");
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, size, size);
  }
  const texture = new THREE.CanvasTexture(canvas);
  texture.minFilter = THREE.LinearFilter;
  texture.magFilter = THREE.LinearFilter;
  return texture;
}

type StarsProps = Required<
  Pick<StarfieldProps, "count" | "colors" | "velocity" | "parallax">
>;

function Stars({ count, colors, velocity, parallax }: StarsProps) {
  const pointsRef = useRef<THREE.Points>(null);
  const mouseRef = useRef({ x: 0, y: 0 });

  const { positions, colorAttr, texture } = useMemo(() => {
    const positions = new Float32Array(count * 3);
    const colorAttr = new Float32Array(count * 3);
    const palette = colors.map((c) => new THREE.Color(c));
    const rand = mulberry32(count * 9301 + 49297);

    for (let i = 0; i < count; i++) {
      positions[i * 3 + 0] = (rand() - 0.5) * DEPTH;
      positions[i * 3 + 1] = (rand() - 0.5) * DEPTH;
      positions[i * 3 + 2] = (rand() - 0.5) * DEPTH;

      const base = palette[Math.floor(rand() * palette.length)];
      const brightness = 0.35 + rand() * 0.65;
      colorAttr[i * 3 + 0] = base.r * brightness;
      colorAttr[i * 3 + 1] = base.g * brightness;
      colorAttr[i * 3 + 2] = base.b * brightness;
    }

    return { positions, colorAttr, texture: buildStarTexture() };
  }, [count, colors]);

  useEffect(() => {
    const handler = (event: PointerEvent) => {
      mouseRef.current.x = (event.clientX / window.innerWidth) * 2 - 1;
      mouseRef.current.y = -((event.clientY / window.innerHeight) * 2 - 1);
    };
    window.addEventListener("pointermove", handler, { passive: true });
    return () => window.removeEventListener("pointermove", handler);
  }, []);

  useFrame((_, delta) => {
    const points = pointsRef.current;
    if (!points) return;

    const arr = points.geometry.attributes.position.array as Float32Array;
    const step = velocity * delta;
    for (let i = 2; i < arr.length; i += 3) {
      arr[i] += step;
      if (arr[i] > HALF_DEPTH) arr[i] -= DEPTH;
    }
    points.geometry.attributes.position.needsUpdate = true;

    const target = mouseRef.current;
    const targetY = target.x * parallax;
    const targetX = -target.y * parallax;
    points.rotation.y += (targetY - points.rotation.y) * 0.04;
    points.rotation.x += (targetX - points.rotation.x) * 0.04;
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
          attach="attributes-color"
          args={[colorAttr, 3]}
          count={count}
        />
      </bufferGeometry>
      <pointsMaterial
        map={texture}
        size={0.45}
        sizeAttenuation
        vertexColors
        transparent
        depthWrite={false}
        blending={THREE.AdditiveBlending}
      />
    </points>
  );
}

export function Starfield({
  count = 1500,
  colors = DEFAULT_COLORS,
  velocity = 5,
  parallax = 0.18,
  className,
}: StarfieldProps) {
  return (
    <div
      className={className}
      style={{
        position: "absolute",
        inset: 0,
        pointerEvents: "none",
      }}
    >
      <Canvas
        camera={{ position: [0, 0, 30], fov: 60 }}
        dpr={[1, 1.5]}
        gl={{
          alpha: true,
          antialias: false,
          powerPreference: "high-performance",
        }}
      >
        <Stars
          count={count}
          colors={colors}
          velocity={velocity}
          parallax={parallax}
        />
      </Canvas>
    </div>
  );
}
