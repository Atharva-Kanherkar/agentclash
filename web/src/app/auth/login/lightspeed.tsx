"use client";

import { useEffect, useRef, useState } from "react";

const FRAGMENT_SHADER = `#version 300 es
precision highp float;
out vec4 O;
uniform float time;
uniform vec2 resolution;
uniform float intensity;
uniform float particleCount;
uniform vec3 colorShift;

#define FC gl_FragCoord.xy
#define R  resolution
#define T  time

float rnd(float a) {
  vec2 p = fract(a * vec2(12.9898, 78.233));
  p += dot(p, p*345.);
  return fract(p.x * p.y);
}

vec3 hue(float a) {
  return colorShift * (.6+.6*cos(6.3*(a)+vec3(0,83,21)));
}

vec3 pattern(vec2 uv) {
  vec3 col = vec3(0.);
  for (float i=.0; i<particleCount; i++) {
    float a = rnd(i);
    vec2 n = vec2(a, fract(a*34.56));
    vec2 p = sin(n*(T+7.) + T*.5);
    float d = dot(uv-p, uv-p);
    col += (intensity * .00125)/d * hue(dot(uv,uv) + i*.125 + T);
  }
  return col;
}

void main(void) {
  vec2 uv = (FC - .5 * R) / min(R.x, R.y);
  vec3 col = vec3(0.);
  float s = 2.4;
  float a = atan(uv.x, uv.y);
  float b = length(uv);
  uv = vec2(a * 5. / 6.28318, .05 / tan(b) + T);
  uv = fract(uv) - .5;
  col += pattern(uv * s);
  O = vec4(col, 1.);
}`;

const VERTEX_SHADER = `#version 300 es
precision highp float;
in vec2 position;
void main(){
  gl_Position = vec4(position, 0.0, 1.0);
}`;

type LightSpeedProps = {
  paused?: boolean;
  speed?: number;
  intensity?: number;
  particleCount?: number;
  colorR?: number;
  colorG?: number;
  colorB?: number;
  quality?: "low" | "medium" | "high";
};

type Uniforms = {
  time: WebGLUniformLocation | null;
  resolution: WebGLUniformLocation | null;
  intensity: WebGLUniformLocation | null;
  particleCount: WebGLUniformLocation | null;
  colorShift: WebGLUniformLocation | null;
};

const qualitySettings = {
  low: { dpr: 0.5, targetFps: 30 },
  medium: { dpr: 1, targetFps: 60 },
  high: { dpr: 1.5, targetFps: 60 },
};

export function LightSpeed({
  paused = false,
  speed = 1,
  intensity = 1,
  particleCount = 20,
  colorR = 1,
  colorG = 1,
  colorB = 1,
  quality = "medium",
}: LightSpeedProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const glRef = useRef<WebGL2RenderingContext | null>(null);
  const programRef = useRef<WebGLProgram | null>(null);
  const vboRef = useRef<WebGLBuffer | null>(null);
  const uniformsRef = useRef<Uniforms>({
    time: null,
    resolution: null,
    intensity: null,
    particleCount: null,
    colorShift: null,
  });
  const rafRef = useRef(0);
  const lastFrameRef = useRef(0);
  const [webglOk, setWebglOk] = useState(true);
  const currentQuality = qualitySettings[quality] ?? qualitySettings.medium;

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const gl = canvas.getContext("webgl2", {
      alpha: false,
      antialias: false,
      depth: false,
      stencil: false,
      powerPreference: "high-performance",
    });

    if (!gl) {
      setWebglOk(false);
      return;
    }

    const compileShader = (type: number, source: string) => {
      const shader = gl.createShader(type);
      if (!shader) throw new Error("Unable to create shader");

      gl.shaderSource(shader, source);
      gl.compileShader(shader);
      if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
        const message = gl.getShaderInfoLog(shader) ?? "Shader compile error";
        gl.deleteShader(shader);
        throw new Error(message);
      }

      return shader;
    };

    const linkProgram = (vertex: WebGLShader, fragment: WebGLShader) => {
      const program = gl.createProgram();
      if (!program) throw new Error("Unable to create WebGL program");

      gl.attachShader(program, vertex);
      gl.attachShader(program, fragment);
      gl.linkProgram(program);
      if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
        const message = gl.getProgramInfoLog(program) ?? "Program link error";
        gl.deleteProgram(program);
        throw new Error(message);
      }

      return program;
    };

    try {
      const vertex = compileShader(gl.VERTEX_SHADER, VERTEX_SHADER);
      const fragment = compileShader(gl.FRAGMENT_SHADER, FRAGMENT_SHADER);
      const program = linkProgram(vertex, fragment);
      programRef.current = program;
      glRef.current = gl;
      gl.useProgram(program);

      const vbo = gl.createBuffer();
      vboRef.current = vbo;
      gl.bindBuffer(gl.ARRAY_BUFFER, vbo);
      gl.bufferData(
        gl.ARRAY_BUFFER,
        new Float32Array([-1, 1, -1, -1, 1, 1, 1, -1]),
        gl.STATIC_DRAW,
      );

      const position = gl.getAttribLocation(program, "position");
      gl.enableVertexAttribArray(position);
      gl.vertexAttribPointer(position, 2, gl.FLOAT, false, 0, 0);

      uniformsRef.current = {
        time: gl.getUniformLocation(program, "time"),
        resolution: gl.getUniformLocation(program, "resolution"),
        intensity: gl.getUniformLocation(program, "intensity"),
        particleCount: gl.getUniformLocation(program, "particleCount"),
        colorShift: gl.getUniformLocation(program, "colorShift"),
      };
      setWebglOk(true);
    } catch {
      setWebglOk(false);
      return;
    }

    const resize = () => {
      const dpr = Math.max(
        1,
        Math.min(window.devicePixelRatio || 1, currentQuality.dpr),
      );
      const cssW = canvas.clientWidth || canvas.parentElement?.clientWidth || 1;
      const cssH =
        canvas.clientHeight || canvas.parentElement?.clientHeight || 1;
      canvas.width = Math.floor(cssW * dpr);
      canvas.height = Math.floor(cssH * dpr);
      gl.viewport(0, 0, canvas.width, canvas.height);
      gl.uniform2f(uniformsRef.current.resolution, canvas.width, canvas.height);
    };

    const observer =
      typeof ResizeObserver === "undefined"
        ? null
        : new ResizeObserver(resize);
    observer?.observe(canvas);
    window.addEventListener("resize", resize);
    resize();

    const start = performance.now();
    const loop = (timestamp: number) => {
      rafRef.current = requestAnimationFrame(loop);
      if (paused) return;

      const delta = timestamp - lastFrameRef.current;
      const targetFrameTime = 1000 / currentQuality.targetFps;
      if (delta < targetFrameTime) return;

      lastFrameRef.current = timestamp - (delta % targetFrameTime);
      const now = (timestamp - start) * 0.001 * (speed || 1);
      const program = programRef.current;
      if (!program) return;

      gl.useProgram(program);
      gl.uniform1f(uniformsRef.current.time, now);
      gl.uniform1f(uniformsRef.current.intensity, intensity);
      gl.uniform1f(uniformsRef.current.particleCount, particleCount);
      gl.uniform3f(uniformsRef.current.colorShift, colorR, colorG, colorB);
      gl.clearColor(0, 0, 0, 1);
      gl.clear(gl.COLOR_BUFFER_BIT);
      gl.drawArrays(gl.TRIANGLE_STRIP, 0, 4);
    };

    rafRef.current = requestAnimationFrame(loop);

    return () => {
      cancelAnimationFrame(rafRef.current);
      observer?.disconnect();
      window.removeEventListener("resize", resize);

      const program = programRef.current;
      if (program) {
        const shaders = gl.getAttachedShaders(program) ?? [];
        shaders.forEach((shader) => gl.deleteShader(shader));
        gl.deleteProgram(program);
      }
      if (vboRef.current) gl.deleteBuffer(vboRef.current);
    };
  }, [
    colorB,
    colorG,
    colorR,
    currentQuality.dpr,
    currentQuality.targetFps,
    intensity,
    particleCount,
    paused,
    speed,
  ]);

  return (
    <div
      aria-label="Lightspeed visual"
      className="relative h-full min-h-[260px] w-full min-w-[100px] overflow-hidden bg-black"
      data-testid="lightspeed-visual"
    >
      {!webglOk && (
        <div className="absolute inset-0 grid place-items-center px-6 text-center text-neutral-200">
          <div className="max-w-md">
            <h2 className="text-xl font-semibold">WebGL not supported</h2>
            <p className="mt-2 text-sm text-white/70">
              Your browser or device does not support WebGL 2.0.
            </p>
          </div>
        </div>
      )}
      <canvas
        ref={canvasRef}
        className="absolute inset-0 block h-full w-full"
        data-testid="lightspeed-canvas"
      />
    </div>
  );
}
