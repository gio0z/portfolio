import React, { useEffect, useRef, useState } from 'react';
import * as THREE from 'three';
import { ArrowDownRight, ExternalLink, Play, Pause, ChevronLeft, ChevronRight } from 'lucide-react';
import type { Project } from '../types';

interface FilmAccordionProps {
  projects: Project[];
}

export const FilmAccordionSection: React.FC<FilmAccordionProps> = ({ projects }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [isPlaying, setIsPlaying] = useState(true);
  const isPlayingRef = useRef(true);

  // References for Three.js state
  const sceneRef = useRef<THREE.Scene | null>(null);
  const rendererRef = useRef<THREE.WebGLRenderer | null>(null);
  const cameraRef = useRef<THREE.PerspectiveCamera | null>(null);
  const meshesRef = useRef<THREE.Mesh[]>([]);
  const scrollOffsetRef = useRef(0);
  const targetOffsetRef = useRef(0);
  const isDraggingRef = useRef(false);
  const startXRef = useRef(0);
  const prevOffsetRef = useRef(0);

  useEffect(() => {
    isPlayingRef.current = isPlaying;
  }, [isPlaying]);

  useEffect(() => {
    if (!canvasRef.current || !containerRef.current || projects.length === 0) return;

    const width = containerRef.current.clientWidth;
    const height = 480;

    // 1. Scene Setup
    const scene = new THREE.Scene();
    sceneRef.current = scene;

    // 2. Camera Setup
    const camera = new THREE.PerspectiveCamera(42, width / height, 0.1, 1000);
    camera.position.set(0, 0, 9.0);
    cameraRef.current = camera;

    // 3. WebGL Renderer with Alpha Transparency
    const renderer = new THREE.WebGLRenderer({
      canvas: canvasRef.current,
      alpha: true,
      antialias: true,
      powerPreference: 'high-performance',
    });
    renderer.setSize(width, height);
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    rendererRef.current = renderer;

    // 4. Lighting
    const ambientLight = new THREE.AmbientLight(0xffffff, 1.4);
    scene.add(ambientLight);

    const centerSpotLight = new THREE.SpotLight(0xffffff, 4, 30, Math.PI / 4, 0.3);
    centerSpotLight.position.set(0, 4, 8);
    scene.add(centerSpotLight);

    const blueRimLight = new THREE.DirectionalLight(0x3b82f6, 1.8);
    blueRimLight.position.set(-6, 3, 5);
    scene.add(blueRimLight);

    // 5. Continuous 3D Film Strip Ribbon Cards
    // Dimensions chosen so adjacent cards connect smoothly
    const cardWidth = 3.35;
    const cardHeight = 2.05;
    const meshes: THREE.Mesh[] = [];

    // Helper: generate translucent film texture with continuous sprocket holes
    const createTransparentFilmTexture = (imgUrl: string, title: string, category: string): THREE.CanvasTexture => {
      const c = document.createElement('canvas');
      c.width = 1024;
      c.height = 640;
      const ctx = c.getContext('2d')!;

      ctx.clearRect(0, 0, c.width, c.height);

      // Translucent film body
      ctx.fillStyle = 'rgba(15, 23, 42, 0.82)';
      ctx.fillRect(0, 0, c.width, c.height);

      // Sprocket border rails (Top & Bottom)
      const barH = 44;
      ctx.fillStyle = 'rgba(6, 9, 17, 0.95)';
      ctx.fillRect(0, 0, c.width, barH);
      ctx.fillRect(0, c.height - barH, c.width, barH);

      // Cutout transparent sprocket holes perfectly centered
      const notchW = 20;
      const notchH = 26;
      const notchGap = 40;
      const totalNotches = Math.floor(c.width / notchGap);

      for (let i = 0; i < totalNotches; i++) {
        const x = 12 + i * notchGap;
        ctx.clearRect(x, 9, notchW, notchH);
        ctx.clearRect(x, c.height - barH + 9, notchW, notchH);
      }

      // Card image area
      ctx.fillStyle = 'rgba(30, 41, 59, 0.9)';
      ctx.fillRect(20, barH + 8, c.width - 40, c.height - (barH * 2) - 16);

      // Category label
      ctx.fillStyle = '#60a5fa';
      ctx.font = '600 22px "Plus Jakarta Sans", sans-serif';
      ctx.fillText(category.toUpperCase(), 40, barH + 44);

      // Project Title
      ctx.fillStyle = '#ffffff';
      ctx.font = 'bold 36px "Plus Jakarta Sans", sans-serif';
      ctx.fillText(title, 40, c.height - barH - 28);

      const texture = new THREE.CanvasTexture(c);

      const img = new Image();
      img.crossOrigin = 'anonymous';
      img.src = imgUrl;
      img.onload = () => {
        ctx.drawImage(img, 20, barH + 8, c.width - 40, c.height - (barH * 2) - 16);

        // Elegant gradient for text readability
        const grad = ctx.createLinearGradient(0, barH, 0, c.height - barH);
        grad.addColorStop(0, 'rgba(15, 23, 42, 0.35)');
        grad.addColorStop(0.45, 'rgba(15, 23, 42, 0.05)');
        grad.addColorStop(1, 'rgba(15, 23, 42, 0.8)');
        ctx.fillStyle = grad;
        ctx.fillRect(20, barH + 8, c.width - 40, c.height - (barH * 2) - 16);

        // Text overlay
        ctx.fillStyle = '#93c5fd';
        ctx.font = '600 22px "Plus Jakarta Sans", sans-serif';
        ctx.fillText(category.toUpperCase(), 40, barH + 44);

        ctx.fillStyle = '#ffffff';
        ctx.font = 'bold 36px "Plus Jakarta Sans", sans-serif';
        ctx.fillText(title, 40, c.height - barH - 28);

        texture.needsUpdate = true;
      };

      return texture;
    };

    const geometry = new THREE.PlaneGeometry(cardWidth, cardHeight, 16, 8);

    projects.forEach((proj) => {
      const texture = createTransparentFilmTexture(proj.image, proj.title, proj.category);
      const material = new THREE.MeshStandardMaterial({
        map: texture,
        transparent: true,
        opacity: 0.96,
        roughness: 0.15,
        metalness: 0.08,
        side: THREE.DoubleSide,
      });

      const mesh = new THREE.Mesh(geometry, material);
      scene.add(mesh);
      meshes.push(mesh);
    });

    meshesRef.current = meshes;

    // 6. Unified Cylindrical Cinema Ribbon Motion
    let animationFrameId: number;
    let autoTime = 0;

    // Circle radius & angular stride for seamless ribbon flow
    const ribbonRadius = 6.2;
    const angularStride = 0.54; // radians between card centers (~31 deg)

    const animate = () => {
      animationFrameId = requestAnimationFrame(animate);

      if (isPlayingRef.current && !isDraggingRef.current) {
        autoTime += 0.0022;
        targetOffsetRef.current += 0.0022;
      }

      scrollOffsetRef.current += (targetOffsetRef.current - scrollOffsetRef.current) * 0.08;

      const total = projects.length;
      const centerPos = scrollOffsetRef.current;

      meshes.forEach((mesh, idx) => {
        let relPos = ((idx - centerPos) % total);
        if (relPos < -total / 2) relPos += total;
        if (relPos > total / 2) relPos -= total;

        // Position on a continuous smooth cylindrical arc
        const theta = relPos * angularStride;
        const x = ribbonRadius * Math.sin(theta);
        const z = ribbonRadius * (Math.cos(theta) - 1);
        const rotY = theta; // Perfectly tangents the arc ribbon

        // Subtle accordion ripple
        const rotZ = Math.sin(relPos * 1.5) * 0.02;

        mesh.position.set(x, 0, z);
        mesh.rotation.set(0, rotY, rotZ);

        // Active center highlight
        const isNearCenter = Math.abs(relPos) < 0.45;
        const mat = mesh.material as THREE.MeshStandardMaterial;
        if (isNearCenter) {
          mat.emissive.setHex(0x1d4ed8);
          mat.emissiveIntensity = 0.28;
          mesh.scale.set(1.04, 1.04, 1.04);
        } else {
          mat.emissive.setHex(0x000000);
          mat.emissiveIntensity = 0.0;
          mesh.scale.set(1.0, 1.0, 1.0);
        }
      });

      const currentActive = (Math.round(centerPos) % total + total) % total;
      setSelectedIndex(currentActive);

      renderer.render(scene, camera);
    };

    animate();

    const handleResize = () => {
      if (!containerRef.current || !renderer || !camera) return;
      const w = containerRef.current.clientWidth;
      camera.aspect = w / height;
      camera.updateProjectionMatrix();
      renderer.setSize(w, height);
    };

    window.addEventListener('resize', handleResize);

    return () => {
      cancelAnimationFrame(animationFrameId);
      window.removeEventListener('resize', handleResize);
      meshes.forEach((m) => {
        m.geometry.dispose();
        if (Array.isArray(m.material)) m.material.forEach((mat) => mat.dispose());
        else m.material.dispose();
      });
      renderer.dispose();
    };
  }, [projects]);

  const handlePointerDown = (e: React.PointerEvent) => {
    isDraggingRef.current = true;
    startXRef.current = e.clientX;
    prevOffsetRef.current = targetOffsetRef.current;
  };

  const handlePointerMove = (e: React.PointerEvent) => {
    if (!isDraggingRef.current) return;
    const deltaX = (e.clientX - startXRef.current) / 220;
    targetOffsetRef.current = prevOffsetRef.current - deltaX;
  };

  const handlePointerUp = () => {
    if (!isDraggingRef.current) return;
    isDraggingRef.current = false;
    targetOffsetRef.current = Math.round(targetOffsetRef.current);
  };

  const jumpToFrame = (idx: number) => {
    targetOffsetRef.current = idx;
  };

  const activeProject = projects[selectedIndex] || projects[0];

  return (
    <section id="projects" className="pt-24 sm:pt-32 pb-24 sm:pb-32 px-4 sm:px-8 max-w-7xl mx-auto overflow-hidden scroll-mt-24">
      {/* Header: Clean & Natural matching the reference style */}
      <div className="flex flex-col md:flex-row md:items-end justify-between mb-12 gap-6">
        <div>
          <div className="inline-flex items-center gap-2 text-xs font-semibold tracking-wider uppercase text-zinc-500 mb-4 font-mono">
            <span className="flex items-center justify-center w-4 h-4 rounded bg-blue-600 text-white">
              <ArrowDownRight className="w-3 h-3" />
            </span>
            <span>Case Studies</span>
          </div>
          <h2 className="text-4xl sm:text-5xl lg:text-6xl font-extrabold tracking-tight text-zinc-900 leading-[1.1]">
            <span>Featured</span> <br />
            <span className="text-zinc-400 font-bold">Engineering Work</span>
          </h2>
        </div>

        {/* Minimal Reel Controls */}
        <div className="flex items-center gap-3 bg-[#18181B] text-white px-4 py-2 rounded-full shadow-lg border border-white/10">
          <div className="text-xs font-mono text-blue-400">
            {selectedIndex + 1} / {projects.length}
          </div>

          <div className="h-3.5 w-px bg-zinc-700 mx-1" />

          <button
            onClick={() => jumpToFrame(selectedIndex - 1)}
            aria-label="Previous"
            className="p-1 hover:text-blue-400 transition-colors cursor-pointer"
          >
            <ChevronLeft className="w-4 h-4" />
          </button>

          <button
            onClick={() => setIsPlaying(!isPlaying)}
            aria-label={isPlaying ? 'Pause' : 'Play'}
            className="p-1 text-zinc-300 hover:text-white transition-colors cursor-pointer"
          >
            {isPlaying ? <Pause className="w-3.5 h-3.5" /> : <Play className="w-3.5 h-3.5" />}
          </button>

          <button
            onClick={() => jumpToFrame(selectedIndex + 1)}
            aria-label="Next"
            className="p-1 hover:text-blue-400 transition-colors cursor-pointer"
          >
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Transparent 3D Continuous Film Ribbon Canvas */}
      <div
        ref={containerRef}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerUp}
        onPointerLeave={handlePointerUp}
        className="w-full relative h-[480px] rounded-[32px] overflow-hidden cursor-grab active:cursor-grabbing select-none bg-transparent"
      >
        <canvas ref={canvasRef} className="w-full h-full block" />
      </div>

      {/* Clean Project Detail Card below the film roll */}
      {activeProject && (
        <div className="mt-8 bg-white rounded-[28px] p-8 sm:p-10 border border-zinc-200/90 shadow-sm hover:shadow-xl transition-all duration-300">
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-center">
            {/* Left: Info */}
            <div className="lg:col-span-8">
              <div className="flex flex-wrap items-center gap-2 mb-3">
                <span className="px-3 py-1 rounded-full bg-blue-50 text-blue-700 text-xs font-semibold font-mono border border-blue-200/60">
                  {activeProject.category}
                </span>
              </div>

              <h3 className="text-2xl sm:text-3xl font-extrabold text-zinc-900 mb-2 tracking-tight">
                {activeProject.title}
              </h3>

              <p className="text-sm font-semibold text-blue-600 mb-4">
                {activeProject.tagline}
              </p>

              <p className="text-sm sm:text-base text-zinc-600 leading-relaxed mb-6 max-w-2xl">
                {activeProject.description}
              </p>

              {/* Tags */}
              <div className="flex flex-wrap gap-2">
                {activeProject.tags.map((t) => (
                  <span
                    key={t}
                    className="px-3 py-1 rounded-md bg-zinc-100 text-zinc-700 text-xs font-mono font-medium"
                  >
                    {t}
                  </span>
                ))}
              </div>
            </div>

            {/* Right: Metrics & Link */}
            <div className="lg:col-span-4 bg-zinc-50 rounded-2xl p-6 border border-zinc-200/80 flex flex-col justify-between">
              <div>
                <div className="text-xs font-mono uppercase text-zinc-400 mb-2 tracking-wider">
                  Verified Metric
                </div>
                <div className="text-sm font-semibold text-zinc-900 bg-white p-3.5 rounded-xl border border-zinc-200/80 mb-6 text-blue-950">
                  ⚡ {activeProject.metrics}
                </div>
              </div>

              <div className="space-y-2.5">
                <a
                  href={activeProject.github_url}
                  target="_blank"
                  rel="noreferrer"
                  className="w-full flex items-center justify-center gap-2 py-3 px-4 rounded-xl bg-zinc-900 hover:bg-zinc-800 text-white font-semibold text-xs transition-colors"
                >
                  <svg className="w-4 h-4 fill-current" viewBox="0 0 24 24">
                    <path fillRule="evenodd" clipRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" />
                  </svg>
                  <span>View Repository on GitHub</span>
                </a>

                {activeProject.demo_url && (
                  <a
                    href={activeProject.demo_url}
                    target="_blank"
                    rel="noreferrer"
                    className="w-full flex items-center justify-center gap-2 py-3 px-4 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs shadow-md shadow-blue-600/20 transition-colors"
                  >
                    <span>Inspect System</span>
                    <ExternalLink className="w-3.5 h-3.5" />
                  </a>
                )}
              </div>
            </div>
          </div>

          {/* Quick Select Buttons */}
          <div className="mt-8 pt-6 border-t border-zinc-100 flex items-center gap-2 overflow-x-auto pb-2">
            {projects.map((p, idx) => (
              <button
                key={p.id}
                onClick={() => jumpToFrame(idx)}
                className={`px-3.5 py-1.5 rounded-lg text-xs font-mono transition-all shrink-0 cursor-pointer ${
                  selectedIndex === idx
                    ? 'bg-blue-600 text-white font-bold shadow-md shadow-blue-600/30'
                    : 'bg-zinc-100 hover:bg-zinc-200 text-zinc-700'
                }`}
              >
                0{idx + 1} {p.title.split(':')[0]}
              </button>
            ))}
          </div>
        </div>
      )}
    </section>
  );
};
