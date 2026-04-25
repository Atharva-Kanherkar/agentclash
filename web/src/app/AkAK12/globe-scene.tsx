"use client";

import { useMemo, useRef } from "react";
import { Canvas, useFrame } from "@react-three/fiber";
import * as THREE from "three";

const INDIA_LAT = 20.5937;
const INDIA_LNG = 78.9629;

function latLngToVector3(
  lat: number,
  lng: number,
  radius: number,
): THREE.Vector3 {
  const phi = (90 - lat) * (Math.PI / 180);
  const theta = (lng + 180) * (Math.PI / 180);
  const x = -radius * Math.sin(phi) * Math.cos(theta);
  const y = radius * Math.cos(phi);
  const z = radius * Math.sin(phi) * Math.sin(theta);
  return new THREE.Vector3(x, y, z);
}

// Deterministic PRNG so the star field is stable across renders
// (React's rules-of-hooks lint forbids Math.random in render).
function mulberry32(seed: number) {
  let t = seed >>> 0;
  return () => {
    t = (t + 0x6d2b79f5) >>> 0;
    let r = t;
    r = Math.imul(r ^ (r >>> 15), r | 1);
    r ^= r + Math.imul(r ^ (r >>> 7), r | 61);
    return ((r ^ (r >>> 14)) >>> 0) / 4294967296;
  };
}

function Stars({ count = 800 }: { count?: number }) {
  const geometry = useMemo(() => {
    const rand = mulberry32(0x5eed1ce);
    const geo = new THREE.BufferGeometry();
    const positions = new Float32Array(count * 3);
    for (let i = 0; i < count; i++) {
      const r = 8 + rand() * 6;
      const theta = rand() * Math.PI * 2;
      const phi = Math.acos(2 * rand() - 1);
      positions[i * 3] = r * Math.sin(phi) * Math.cos(theta);
      positions[i * 3 + 1] = r * Math.sin(phi) * Math.sin(theta);
      positions[i * 3 + 2] = r * Math.cos(phi);
    }
    geo.setAttribute("position", new THREE.BufferAttribute(positions, 3));
    return geo;
  }, [count]);

  return (
    <points geometry={geometry}>
      <pointsMaterial
        size={0.025}
        color="#cfd6ff"
        sizeAttenuation
        transparent
        opacity={0.7}
      />
    </points>
  );
}

function IndiaMarker() {
  const haloRef = useRef<THREE.Mesh>(null);
  const ringRef = useRef<THREE.Mesh>(null);

  const position = useMemo(
    () => latLngToVector3(INDIA_LAT, INDIA_LNG, 1.001),
    [],
  );

  // Orient the marker so its local +Y axis points radially outward.
  const quaternion = useMemo(() => {
    const up = new THREE.Vector3(0, 1, 0);
    const target = position.clone().normalize();
    return new THREE.Quaternion().setFromUnitVectors(up, target);
  }, [position]);

  useFrame((state) => {
    const t = state.clock.elapsedTime;
    if (haloRef.current) {
      const pulse = 1 + Math.sin(t * 2.5) * 0.15;
      haloRef.current.scale.setScalar(pulse);
    }
    if (ringRef.current) {
      const expand = 1 + ((t * 0.6) % 1);
      const fade = 1 - ((t * 0.6) % 1);
      ringRef.current.scale.setScalar(expand);
      const mat = ringRef.current.material as THREE.MeshBasicMaterial;
      mat.opacity = 0.55 * fade;
    }
  });

  return (
    <group position={position} quaternion={quaternion}>
      <mesh>
        <sphereGeometry args={[0.018, 16, 16]} />
        <meshBasicMaterial color="#FFB347" />
      </mesh>
      <mesh ref={haloRef}>
        <sphereGeometry args={[0.034, 16, 16]} />
        <meshBasicMaterial color="#FF9933" transparent opacity={0.45} />
      </mesh>
      <mesh ref={ringRef} rotation={[Math.PI / 2, 0, 0]} position={[0, 0.001, 0]}>
        <ringGeometry args={[0.04, 0.06, 48]} />
        <meshBasicMaterial
          color="#FF9933"
          transparent
          opacity={0.55}
          side={THREE.DoubleSide}
        />
      </mesh>
    </group>
  );
}

function Globe() {
  const groupRef = useRef<THREE.Group>(null);

  useFrame((_, delta) => {
    if (groupRef.current) {
      groupRef.current.rotation.y += delta * 0.07;
    }
  });

  return (
    <group ref={groupRef} rotation={[0.35, 0, 0]}>
      <mesh>
        <sphereGeometry args={[1, 96, 96]} />
        <meshStandardMaterial
          color="#0b1024"
          roughness={0.85}
          metalness={0.1}
          emissive="#1d2452"
          emissiveIntensity={0.18}
        />
      </mesh>

      <mesh>
        <sphereGeometry args={[1.0015, 48, 32]} />
        <meshBasicMaterial
          color="#5263c7"
          wireframe
          transparent
          opacity={0.18}
        />
      </mesh>

      <IndiaMarker />
    </group>
  );
}

function Atmosphere() {
  return (
    <mesh>
      <sphereGeometry args={[1.18, 96, 96]} />
      <shaderMaterial
        transparent
        depthWrite={false}
        side={THREE.BackSide}
        vertexShader={`
          varying vec3 vNormal;
          void main() {
            vNormal = normalize(normalMatrix * normal);
            gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
          }
        `}
        fragmentShader={`
          varying vec3 vNormal;
          void main() {
            float intensity = pow(0.62 - dot(vNormal, vec3(0.0, 0.0, 1.0)), 2.4);
            gl_FragColor = vec4(0.40, 0.55, 1.0, 1.0) * intensity;
          }
        `}
      />
    </mesh>
  );
}

export function GlobeScene() {
  return (
    <Canvas
      camera={{ position: [0.4, 0.6, 2.6], fov: 42 }}
      gl={{ antialias: true, alpha: true }}
      dpr={[1, 2]}
    >
      <ambientLight intensity={0.35} />
      <directionalLight position={[5, 3, 5]} intensity={1.4} color="#ffffff" />
      <directionalLight position={[-4, -2, -3]} intensity={0.45} color="#7a8cff" />
      <Stars />
      <Atmosphere />
      <Globe />
    </Canvas>
  );
}
