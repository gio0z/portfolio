import React, { useEffect, useRef, useState } from 'react';
import * as THREE from 'three';
import { ExternalLink, Play, Pause, ChevronLeft, ChevronRight, Film } from 'lucide-react';
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
    const camera = new THREE.PerspectiveCamera(45, width / height, 0.1, 1000);
    camera.position.set(0, 0, 8.5);
    cameraRef.current = camera;

    // 3. WebGL Renderer
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
    const ambientLight = new THREE.AmbientLight(0xffffff, 1.2);
    scene.add(ambientLight);

    const pointLight = new THREE.PointLight(0x3b82f6, 3, 20);
    pointLight.position.set(0, 3, 6);
    scene.add(pointLight);

    const blueRimLight = new THREE.DirectionalLight(0x2563eb, 1.5);
    blueRimLight.position.set(-5, 5, 4);
    scene.add(blueRimLight);

    // 5. Build 3D Film Strip Panels (Canvas-based textures with sprocket holes)
    const cardWidth = 3.2;
    const cardHeight = 2.0;
    const spacing = 3.7;
    const meshes: THREE.Mesh[] = [];

    // Helper to generate a film frame texture with sprocket holes
    const createFilmTexture = (imgUrl: string, title: string, index: number): THREE.CanvasTexture => {
      const c = document.createElement('canvas');
      c.width = 1024;
      c.height = 640;
      const ctx = c.getContext('2d')!;

      // Background celluloid film color
      ctx.fillStyle = '#10141e';
      ctx.fillRect(0, 0, c.width, c.height);

      // Sprocket holes border bars (Top & Bottom)
      const sprocketBarHeight = 54;
      ctx.fillStyle = '#060911';
      ctx.fillRect(0, 0, c.width, sprocketBarHeight);
      ctx.fillRect(0, c.height - sprocketBarHeight, c.width, sprocketBarHeight);

      // Draw sprocket holes
      const holeWidth = 26;
      const holeHeight = 36;
      const holeGap = 42;
      const numHoles = Math.floor(c.width / holeGap);

      ctx.fillStyle = '#f8f9fa';
      for (let i = 0; i < numHoles; i++) {
        const x = 16 + i * holeGap;
        // Top hole
        ctx.beginPath();
        ctx.roundRect(x, 9, holeWidth, holeHeight, 6);
        ctx.fill();

        // Bottom hole
        ctx.beginPath();
        ctx.roundRect(x, c.height - sprocketBarHeight + 9, holeWidth, holeHeight, 6);
        ctx.fill();
      }

      // Draw inner film frame placeholder while image loads
      ctx.fillStyle = '#181f33';
      ctx.fillRect(32, sprocketBarHeight + 12, c.width - 64, c.height - (sprocketBarHeight * 2) - 24);

      // Film frame counter
      ctx.fillStyle = '#3b82f6';
      ctx.font = 'bold 22px "JetBrains Mono", monospace';
      ctx.fillText(`FRAME 0${index + 1} // 35MM REEL`, 48, sprocketBarHeight + 46);

      ctx.fillStyle = '#ffffff';
      ctx.font = 'bold 36px "Plus Jakarta Sans", sans-serif';
      ctx.fillText(title, 48, c.height - sprocketBarHeight - 32);

      const texture = new THREE.CanvasTexture(c);

      // Load background image onto canvas
      const img = new Image();
      img.crossOrigin = 'anonymous';
      img.src = imgUrl;
      img.onload = () => {
        // Draw image in central active area
        ctx.drawImage(img, 32, sprocketBarHeight + 12, c.width - 64, c.height - (sprocketBarHeight * 2) - 24);
        
        // Add subtle film grain / dark gradient vignette
        const gradient = ctx.createLinearGradient(0, sprocketBarHeight, 0, c.height - sprocketBarHeight);
        gradient.addColorStop(0, 'rgba(6, 9, 17, 0.4)');
        gradient.addColorStop(0.5, 'rgba(6, 9, 17, 0.1)');
        gradient.addColorStop(1, 'rgba(6, 9, 17, 0.7)');
        ctx.fillStyle = gradient;
        ctx.fillRect(32, sprocketBarHeight + 12, c.width - 64, c.height - (sprocketBarHeight * 2) - 24);

        // Re-render labels on top of image
        ctx.fillStyle = '#38bdf8';
        ctx.font = 'bold 24px "JetBrains Mono", monospace';
        ctx.fillText(`FRAME 0${index + 1} // GIO0Z / ${projects[index]?.category || 'SYSTEMS'}`, 52, sprocketBarHeight + 48);

        ctx.fillStyle = '#ffffff';
        ctx.font = 'bold 38px "Plus Jakarta Sans", sans-serif';
        ctx.fillText(title, 52, c.height - sprocketBarHeight - 34);

        texture.needsUpdate = true;
      };

      return texture;
    };

    // Geometry: slightly segmented plane to allow accordion curvature
    const geometry = new THREE.PlaneGeometry(cardWidth, cardHeight, 16, 8);

    projects.forEach((proj, idx) => {
      const texture = createFilmTexture(proj.image, proj.title, idx);
      const material = new THREE.MeshStandardMaterial({
        map: texture,
        roughness: 0.25,
        metalness: 0.1,
        side: THREE.DoubleSide,
      });

      const mesh = new THREE.Mesh(geometry, material);
      mesh.userData = { index: idx };
      scene.add(mesh);
      meshes.push(mesh);
    });

    meshesRef.current = meshes;

    // 6. Animation Loop (Accordion Film Curve)
    let animationFrameId: number;
    let autoTime = 0;

    const animate = () => {
      animationFrameId = requestAnimationFrame(animate);

      // Auto scroll if playing and not user dragging
      if (isPlayingRef.current && !isDraggingRef.current) {
        autoTime += 0.003;
        targetOffsetRef.current += 0.003;
      }

      // Smooth damping lerp
      scrollOffsetRef.current += (targetOffsetRef.current - scrollOffsetRef.current) * 0.08;

      const total = projects.length;
      const centerPos = scrollOffsetRef.current;

      // Position each film card in 3D space with an accordion wave
      meshes.forEach((mesh, idx) => {
        let relPos = ((idx - centerPos) % total);
        if (relPos < -total / 2) relPos += total;
        if (relPos > total / 2) relPos -= total;

        const x = relPos * spacing;
        
        // Accordion 3D depth wave: closer when near center, folds backwards when away
        const z = -Math.abs(relPos) * 0.9 + (Math.abs(relPos) < 0.6 ? 0.3 : 0);
        
        // Accordion rotation (like folded film reel / paper accordion)
        const rotY = -relPos * 0.28;
        const rotZ = Math.sin(relPos * 0.8) * 0.04;

        mesh.position.set(x, 0, z);
        mesh.rotation.set(0, rotY, rotZ);

        // Highlight selected
        const isNearCenter = Math.abs(relPos) < 0.5;
        const mat = mesh.material as THREE.MeshStandardMaterial;
        if (isNearCenter) {
          mat.emissive.setHex(0x1d4ed8);
          mat.emissiveIntensity = 0.25;
        } else {
          mat.emissive.setHex(0x000000);
          mat.emissiveIntensity = 0.0;
        }
      });

      // Update selected index in React state when active frame changes
      const currentActive = (Math.round(centerPos) % total + total) % total;
      setSelectedIndex(currentActive);

      renderer.render(scene, camera);
    };

    animate();

    // 7. Mouse/Touch Drag Interactivity
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

  // Drag handlers
  const handlePointerDown = (e: React.PointerEvent) => {
    isDraggingRef.current = true;
    startXRef.current = e.clientX;
    prevOffsetRef.current = targetOffsetRef.current;
  };

  const handlePointerMove = (e: React.PointerEvent) => {
    if (!isDraggingRef.current) return;
    const deltaX = (e.clientX - startXRef.current) / 240;
    targetOffsetRef.current = prevOffsetRef.current - deltaX;
  };

  const handlePointerUp = () => {
    if (!isDraggingRef.current) return;
    isDraggingRef.current = false;
    // Snap to nearest frame
    targetOffsetRef.current = Math.round(targetOffsetRef.current);
  };

  const jumpToFrame = (idx: number) => {
    targetOffsetRef.current = idx;
  };

  const activeProject = projects[selectedIndex] || projects[0];

  return (
    <section id="projects" className="py-24 sm:py-32 px-4 sm:px-8 max-w-7xl mx-auto overflow-hidden">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-end justify-between mb-10 gap-6">
        <div>
          <div className="inline-flex items-center gap-2 text-xs font-semibold tracking-wider uppercase text-zinc-500 mb-4 font-mono">
            <span className="flex items-center justify-center w-4 h-4 rounded bg-blue-600 text-white">
              <Film className="w-3 h-3" />
            </span>
            <span>Real Repositories from GitHub /gio0z</span>
          </div>
          <h2 className="text-4xl sm:text-5xl lg:text-6xl font-extrabold tracking-tight text-zinc-900 leading-[1.1]">
            <span>Interactive 3D</span> <br />
            <span className="text-zinc-400 font-bold">Film Reel Showcase</span>
          </h2>
        </div>

        {/* Film Controls Bar */}
        <div className="flex items-center gap-3 bg-[#18181B] text-white px-5 py-2.5 rounded-full shadow-lg border border-white/10">
          <div className="flex items-center gap-2 text-xs font-mono text-blue-400">
            <span className="w-2 h-2 rounded-full bg-blue-500 animate-pulse"></span>
            <span>FRAME {selectedIndex + 1} / {projects.length}</span>
          </div>

          <div className="h-4 w-px bg-zinc-700 mx-1" />

          <button
            onClick={() => jumpToFrame(selectedIndex - 1)}
            aria-label="Previous frame"
            className="p-1 hover:text-blue-400 transition-colors cursor-pointer"
          >
            <ChevronLeft className="w-4 h-4" />
          </button>

          <button
            onClick={() => setIsPlaying(!isPlaying)}
            aria-label={isPlaying ? 'Pause film' : 'Play film'}
            className="p-1 text-zinc-300 hover:text-white transition-colors cursor-pointer"
          >
            {isPlaying ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
          </button>

          <button
            onClick={() => jumpToFrame(selectedIndex + 1)}
            aria-label="Next frame"
            className="p-1 hover:text-blue-400 transition-colors cursor-pointer"
          >
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* 3D Three.js Film Strip Canvas Container */}
      <div
        ref={containerRef}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerUp}
        onPointerLeave={handlePointerUp}
        className="w-full relative h-[480px] bg-[#0c101c] rounded-[32px] overflow-hidden border border-zinc-800 shadow-2xl cursor-grab active:cursor-grabbing select-none"
      >
        <canvas ref={canvasRef} className="w-full h-full block" />

        {/* Cinematic Film Overlay Badges */}
        <div className="absolute top-4 left-6 pointer-events-none flex items-center gap-3">
          <span className="px-3 py-1 rounded-full bg-black/60 backdrop-blur-md text-[11px] font-mono text-blue-400 border border-blue-500/30">
            35MM CELLULOID REEL
          </span>
          <span className="px-3 py-1 rounded-full bg-black/60 backdrop-blur-md text-[11px] font-mono text-zinc-400 border border-white/10">
            DRAG OR CLICK TO ROTATE
          </span>
        </div>

        <div className="absolute bottom-4 right-6 pointer-events-none text-right">
          <span className="text-[11px] font-mono text-zinc-500 tracking-wider">
            THREE.JS ACCORDION CAMERA // 60 FPS
          </span>
        </div>
      </div>

      {/* Synchronized Project Synoptic Card (Active Film Frame Detail) */}
      {activeProject && (
        <div className="mt-8 bg-white rounded-[28px] p-8 sm:p-10 border border-zinc-200 shadow-xl transition-all duration-300">
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-center">
            {/* Left: Info & Scope */}
            <div className="lg:col-span-8">
              <div className="flex flex-wrap items-center gap-2 mb-3">
                <span className="px-3 py-1 rounded-full bg-blue-100 text-blue-800 text-xs font-semibold">
                  {activeProject.category}
                </span>
                <span className="text-xs font-mono text-zinc-500">
                  Frame 0{selectedIndex + 1} of {projects.length}
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

              {/* Tech stack badges */}
              <div className="flex flex-wrap gap-2 mb-4">
                {activeProject.tags.map((t) => (
                  <span key={t} className="px-3 py-1 rounded-md bg-zinc-100 text-zinc-800 text-xs font-mono font-medium">
                    {t}
                  </span>
                ))}
              </div>
            </div>

            {/* Right: Metrics & Actions */}
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

          {/* Quick Select Thumbnails Strip */}
          <div className="mt-8 pt-6 border-t border-zinc-100 flex items-center gap-3 overflow-x-auto pb-2">
            <span className="text-xs font-mono text-zinc-400 shrink-0 mr-2">FRAME SELECTOR:</span>
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
