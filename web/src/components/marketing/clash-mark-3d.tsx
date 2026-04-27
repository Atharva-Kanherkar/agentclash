"use client";

/*
 * ClashMark3D — the AgentClash logo as two extruded glass prisms that face
 * each other and run the same wind-up → clash → recoil cycle as the SVG mark.
 * MeshTransmissionMaterial gives real refraction + chromatic aberration; the
 * group rotates slowly so the depth reads even when the clash is settled.
 */

import { Canvas, useFrame } from "@react-three/fiber";
import { MeshTransmissionMaterial } from "@react-three/drei";
import { useMemo, useRef } from "react";
import * as THREE from "three";

const CLASH_PERIOD = 2.5;
const HOME_X = 1.05;
const WIND_UP = 0.7;
const IMPACT_OVERSHOOT = 0.13;

function smoothstep(t: number) {
  return t * t * (3 - 2 * t);
}

// Mirrors the SVG keyframes (translateX -24 → 0 → +4 → 0) in normalized world
// units. Returns the per-prism x-offset added to the home position.
function clashOffset(cyclePhase: number) {
  if (cyclePhase < 0.18) {
    const p = cyclePhase / 0.18;
    return -WIND_UP * (1 - smoothstep(p));
  }
  if (cyclePhase < 0.23) {
    const p = (cyclePhase - 0.18) / 0.05;
    return IMPACT_OVERSHOOT * smoothstep(p);
  }
  if (cyclePhase < 0.3) {
    const p = (cyclePhase - 0.23) / 0.07;
    return IMPACT_OVERSHOOT * (1 - smoothstep(p));
  }
  return 0;
}

function buildPrismGeometry(): THREE.BufferGeometry {
  // Triangle with apex along +X. Shift by -centroid so the pivot sits at the
  // triangle's center of mass; rotations and clash translations then look
  // anchored rather than swinging from the base.
  const cx = 1 / 3;
  const s = new THREE.Shape();
  s.moveTo(-1 + cx, 0.78);
  s.lineTo(1 - cx, 0);
  s.lineTo(-1 + cx, -0.78);
  s.closePath();
  const geom = new THREE.ExtrudeGeometry(s, {
    depth: 0.5,
    bevelEnabled: true,
    bevelSegments: 2,
    bevelThickness: 0.04,
    bevelSize: 0.04,
    curveSegments: 1,
  });
  geom.translate(0, 0, -0.25);
  geom.computeVertexNormals();
  return geom;
}

function ClashPrism({
  side,
  animated,
}: {
  side: "left" | "right";
  animated: boolean;
}) {
  const meshRef = useRef<THREE.Mesh>(null);
  const geometry = useMemo(() => buildPrismGeometry(), []);
  const dir = side === "left" ? -1 : 1;
  // Right prism is rotated 180° around Y so its apex points back toward the
  // origin; both prisms therefore face inward.
  const baseYaw = side === "left" ? 0 : Math.PI;

  useFrame((state) => {
    const mesh = meshRef.current;
    if (!mesh) return;
    const t = state.clock.elapsedTime;
    const phase = animated ? (t % CLASH_PERIOD) / CLASH_PERIOD : 0;
    const offset = animated ? clashOffset(phase) : 0;
    mesh.position.x = HOME_X * dir + offset * dir;
    mesh.rotation.y = baseYaw;
  });

  return (
    <mesh ref={meshRef} geometry={geometry}>
      <MeshTransmissionMaterial
        thickness={0.6}
        roughness={0.06}
        transmission={1}
        ior={1.5}
        chromaticAberration={0.05}
        anisotropy={0.1}
        distortion={0.18}
        distortionScale={0.4}
        temporalDistortion={0.08}
        backside
        samples={6}
        resolution={512}
        color="#ffffff"
        attenuationColor="#cfe2ff"
        attenuationDistance={2.5}
      />
    </mesh>
  );
}

function SceneTilt({ children }: { children: React.ReactNode }) {
  const groupRef = useRef<THREE.Group>(null);
  useFrame((state) => {
    const g = groupRef.current;
    if (!g) return;
    // Slow continuous yaw so the depth always reads. Tiny pitch wobble too.
    const t = state.clock.elapsedTime;
    g.rotation.y = Math.sin(t * 0.25) * 0.18;
    g.rotation.x = Math.sin(t * 0.15) * 0.06;
  });
  return <group ref={groupRef}>{children}</group>;
}

export function ClashMark3D({
  animated = false,
  className,
}: {
  animated?: boolean;
  className?: string;
}) {
  return (
    <div className={className} style={{ width: "100%", height: "100%" }}>
      <Canvas
        camera={{ position: [0, 0, 4.2], fov: 42 }}
        dpr={[1, 1.5]}
        gl={{
          alpha: true,
          antialias: true,
          powerPreference: "high-performance",
        }}
      >
        <ambientLight intensity={0.45} />
        <directionalLight position={[4, 4, 5]} intensity={2.2} />
        <directionalLight
          position={[-5, -2, 3]}
          intensity={1.2}
          color="#7eb8e6"
        />
        <directionalLight
          position={[0, 3, -3]}
          intensity={0.9}
          color="#ffd4a8"
        />
        <SceneTilt>
          <ClashPrism side="left" animated={animated} />
          <ClashPrism side="right" animated={animated} />
        </SceneTilt>
      </Canvas>
    </div>
  );
}
