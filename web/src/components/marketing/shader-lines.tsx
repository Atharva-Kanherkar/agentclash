"use client";

/*
 * ShaderLines — vendored from a Framer marketplace component (Launchly's
 * Shader Background, free tier). Stripped of Framer-only imports
 * (addPropertyControls, ControlType, defaultProps) and the CDN-loaded copy
 * of three.js v89; we use the locally-installed three package instead.
 *
 * Renders a full-rect WebGL shader of vertical, mosaic-quantized light
 * streaks that pulse between two colors. Intended as a non-interactive
 * backdrop — the canvas itself is pointer-events: none so it never
 * intercepts clicks meant for content layered above it.
 */

import { useEffect, useRef } from "react";
import * as THREE from "three";

export type ShaderLinesProps = {
  /** Time delta added to the shader's `time` uniform per frame. */
  animationSpeed?: number;
  /** Multiplier on streak brightness. Lower = subtler. */
  colorIntensity?: number;
  /** Mosaic cell scale; smaller x = wider vertical streaks. */
  mosaicScale?: { x: number; y: number };
  colorA?: string;
  colorB?: string;
  /** Painted as the shader's clear color; fades into the page below. */
  backgroundColor?: string;
  /**
   * 0 = streaks cover the full rect (default).
   * 1 = full mask: center horizontal band is clear, streaks only on the
   * left and right edges. Useful when content sits in the middle.
   */
  centerFade?: number;
  className?: string;
  style?: React.CSSProperties;
};

const VERTEX_SHADER = /* glsl */ `
  void main() {
    gl_Position = vec4(position, 1.0);
  }
`;

const FRAGMENT_SHADER = /* glsl */ `
  #define PI 3.14159265359
  precision highp float;

  uniform vec2 resolution;
  uniform float time;
  uniform vec2 mosaicScale;
  uniform float colorIntensity;
  uniform vec3 colorA;
  uniform vec3 colorB;
  uniform vec3 bgColor;
  uniform float centerFade;

  float random(in float x) { return fract(sin(x) * 1e4); }
  float random(vec2 st) {
    return fract(sin(dot(st.xy, vec2(12.9898, 78.233))) * 43758.5453123);
  }

  void main(void) {
    vec2 uv = (gl_FragCoord.xy * 2.0 - resolution.xy) / min(resolution.x, resolution.y);
    // Preserve pre-quantization uv for the side mask so the fade is smooth
    // even though the streak pattern itself is mosaic-quantized.
    vec2 origUv = uv;
    vec2 vScreenSize = vec2(256.0, 256.0);

    uv.x = floor(uv.x * vScreenSize.x / mosaicScale.x) / (vScreenSize.x / mosaicScale.x);
    uv.y = floor(uv.y * vScreenSize.y / mosaicScale.y) / (vScreenSize.y / mosaicScale.y);

    float t = time * 0.06 + random(uv.x) * 0.4;
    float lineWidth = 0.0008;

    float intensity = 0.0;
    for (int j = 0; j < 3; j++) {
      for (int i = 0; i < 5; i++) {
        intensity +=
          lineWidth * float(i * i) /
          abs(fract(t - 0.01 * float(j) + float(i) * 0.01) - length(uv));
      }
    }

    // Side mask: 0 in the center, ramping to 1 at the horizontal edges.
    // centerFade is the lerp amount between "no mask" and "full mask".
    float sideMask = mix(1.0, smoothstep(0.4, 1.0, abs(origUv.x)), centerFade);
    intensity *= sideMask;

    vec3 lineColor = mix(colorA, colorB, 0.5 + 0.5 * sin(time * 0.5 + uv.x * PI));
    vec3 finalColor = bgColor + intensity * lineColor * colorIntensity;
    gl_FragColor = vec4(finalColor, 1.0);
  }
`;

export function ShaderLines({
  animationSpeed = 0.04,
  colorIntensity = 0.65,
  mosaicScale = { x: 4.0, y: 2.0 },
  colorA = "#7eb8e6",
  colorB = "#ff63b8",
  backgroundColor = "#060606",
  centerFade = 0,
  className,
  style,
}: ShaderLinesProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  // Latest values mirrored into refs so the rAF loop reads fresh props
  // without tearing down the WebGL context on every prop change.
  const speedRef = useRef(animationSpeed);
  const intensityRef = useRef(colorIntensity);
  const centerFadeRef = useRef(centerFade);

  useEffect(() => {
    speedRef.current = animationSpeed;
    intensityRef.current = colorIntensity;
    centerFadeRef.current = centerFade;
  }, [animationSpeed, colorIntensity, centerFade]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const camera = new THREE.Camera();
    camera.position.z = 1;
    const scene = new THREE.Scene();
    const geometry = new THREE.PlaneGeometry(2, 2);

    const uniforms = {
      time: { value: 1.0 },
      resolution: { value: new THREE.Vector2() },
      mosaicScale: {
        value: new THREE.Vector2(mosaicScale.x, mosaicScale.y),
      },
      colorIntensity: { value: colorIntensity },
      colorA: { value: new THREE.Color(colorA) },
      colorB: { value: new THREE.Color(colorB) },
      bgColor: { value: new THREE.Color(backgroundColor) },
      centerFade: { value: centerFade },
    };

    const material = new THREE.ShaderMaterial({
      uniforms,
      vertexShader: VERTEX_SHADER,
      fragmentShader: FRAGMENT_SHADER,
    });

    const mesh = new THREE.Mesh(geometry, material);
    scene.add(mesh);

    const renderer = new THREE.WebGLRenderer({
      alpha: true,
      antialias: false,
      powerPreference: "high-performance",
    });
    const dpr = Math.min(
      typeof window === "undefined" ? 1 : window.devicePixelRatio || 1,
      1.5,
    );
    renderer.setPixelRatio(dpr);
    renderer.setClearColor(0x000000, 0);
    renderer.domElement.style.display = "block";
    renderer.domElement.style.width = "100%";
    renderer.domElement.style.height = "100%";
    container.appendChild(renderer.domElement);

    const resize = () => {
      const rect = container.getBoundingClientRect();
      if (rect.width === 0 || rect.height === 0) return;
      renderer.setSize(rect.width, rect.height, false);
      uniforms.resolution.value.x = rect.width * dpr;
      uniforms.resolution.value.y = rect.height * dpr;
    };
    resize();

    const ro =
      typeof ResizeObserver === "undefined" ? null : new ResizeObserver(resize);
    ro?.observe(container);
    window.addEventListener("resize", resize);

    let rafId = 0;
    const animate = () => {
      rafId = requestAnimationFrame(animate);
      uniforms.time.value += speedRef.current;
      uniforms.colorIntensity.value = intensityRef.current;
      uniforms.centerFade.value = centerFadeRef.current;
      renderer.render(scene, camera);
    };
    animate();

    return () => {
      cancelAnimationFrame(rafId);
      ro?.disconnect();
      window.removeEventListener("resize", resize);
      renderer.dispose();
      geometry.dispose();
      material.dispose();
      if (renderer.domElement.parentNode === container) {
        container.removeChild(renderer.domElement);
      }
    };
    // Re-create the WebGL context only when something structural changes;
    // animationSpeed and colorIntensity flow through refs above.
  }, [
    backgroundColor,
    centerFade,
    colorA,
    colorB,
    colorIntensity,
    mosaicScale.x,
    mosaicScale.y,
  ]);

  return (
    <div
      ref={containerRef}
      aria-hidden
      className={className}
      style={{
        width: "100%",
        height: "100%",
        position: "relative",
        backgroundColor,
        pointerEvents: "none",
        ...style,
      }}
    />
  );
}
