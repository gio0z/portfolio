import React, { useState, useEffect, useCallback } from 'react';
import { ArrowDownRight, ChevronLeft, ChevronRight, ExternalLink } from 'lucide-react';
import type { Project } from '../types';

interface CoverflowProps {
  projects: Project[];
}

export const CoverflowSection: React.FC<CoverflowProps> = ({ projects }) => {
  const [currentIndex, setCurrentIndex] = useState(0);
  const [touchStartX, setTouchStartX] = useState<number | null>(null);

  const total = projects.length;

  const next = useCallback(() => {
    setCurrentIndex((prev) => (prev + 1) % total);
  }, [total]);

  const prev = useCallback(() => {
    setCurrentIndex((prev) => (prev - 1 + total) % total);
  }, [total]);

  // Keyboard navigation (ArrowLeft, ArrowRight)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'ArrowLeft') prev();
      if (e.key === 'ArrowRight') next();
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [next, prev]);

  // Touch Swipe Handlers
  const handleTouchStart = (e: React.TouchEvent) => {
    setTouchStartX(e.touches[0].clientX);
  };

  const handleTouchEnd = (e: React.TouchEvent) => {
    if (touchStartX === null) return;
    const deltaX = e.changedTouches[0].clientX - touchStartX;
    if (deltaX > 40) prev();
    else if (deltaX < -40) next();
    setTouchStartX(null);
  };

  // Color palette progression based on distance from center (matching the screenshot)
  const getCardBg = (offset: number) => {
    const abs = Math.abs(offset);
    if (abs === 0) return '#2A4B75'; // Deep denim / slate blue
    if (abs === 1) return '#3E6596'; // Medium royal slate
    if (abs === 2) return '#688CB8'; // Soft cornflower blue
    return '#A3BCD6'; // Pale lavender blue fade
  };

  const activeProject = projects[currentIndex] || projects[0];

  return (
    <section id="projects" className="pt-24 sm:pt-32 pb-24 sm:pb-32 px-4 sm:px-8 max-w-7xl mx-auto overflow-hidden scroll-mt-28">
      {/* Header: Clean Editorial Style */}
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

        {/* Counter Pill */}
        <div className="flex items-center gap-3 bg-[#18181B] text-white px-5 py-2.5 rounded-full shadow-lg border border-white/10">
          <div className="text-xs font-mono text-blue-400">
            0{currentIndex + 1} <span className="text-zinc-500">/ 0{total}</span>
          </div>
          <div className="h-3 w-px bg-zinc-700 mx-1" />
          <span className="text-xs font-medium text-zinc-300">3D Coverflow</span>
        </div>
      </div>

      {/* 3D Coverflow Viewport Container (matching the reference stage) */}
      <div
        onTouchStart={handleTouchStart}
        onTouchEnd={handleTouchEnd}
        className="relative w-full bg-white/95 rounded-[32px] sm:rounded-[40px] p-6 sm:p-12 shadow-[0_30px_90px_-20px_rgba(0,0,0,0.09)] border border-zinc-200/80 flex items-center justify-center min-h-[480px] sm:min-h-[560px] overflow-hidden select-none"
      >
        {/* Left Navigation Chevron Button */}
        <button
          onClick={prev}
          aria-label="Previous card"
          className="absolute left-3 sm:left-8 z-40 p-3 rounded-full text-zinc-800 hover:text-blue-600 hover:bg-zinc-100/80 transition-all cursor-pointer group active:scale-95"
        >
          <ChevronLeft className="w-8 h-8 sm:w-10 sm:h-10 stroke-[1.5] group-hover:-translate-x-0.5 transition-transform" />
        </button>

        {/* Right Navigation Chevron Button */}
        <button
          onClick={next}
          aria-label="Next card"
          className="absolute right-3 sm:right-8 z-40 p-3 rounded-full text-zinc-800 hover:text-blue-600 hover:bg-zinc-100/80 transition-all cursor-pointer group active:scale-95"
        >
          <ChevronRight className="w-8 h-8 sm:w-10 sm:h-10 stroke-[1.5] group-hover:translate-x-0.5 transition-transform" />
        </button>

        {/* 3D Stack Stage */}
        <div
          className="relative w-full h-[380px] sm:h-[450px] flex items-center justify-center"
          style={{ perspective: '1000px', transformStyle: 'preserve-3d' }}
        >
          {projects.map((proj, idx) => {
            let offset = idx - currentIndex;

            // Circular wrap-around
            if (offset > total / 2) offset -= total;
            if (offset < -total / 2) offset += total;

            const absOffset = Math.abs(offset);
            const isVisible = absOffset <= 2;
            if (!isVisible) return null;

            // 3D Matrix Math (replicating reference image depth, angles, scale, and occlusion)
            const isMobile = typeof window !== 'undefined' && window.innerWidth < 640;
            const xStride = isMobile ? 95 : 160;
            const x = offset * xStride;
            const z = offset === 0 ? 50 : -absOffset * 100;
            const rotY = offset === 0 ? 0 : offset < 0 ? 32 : -32; // Inward 3D tilt
            const scale = offset === 0 ? 1.02 : 1 - absOffset * 0.08;
            const opacity = 1 - absOffset * 0.15;
            const zIndex = 30 - absOffset * 5;
            const bgColor = getCardBg(offset);
            const isActive = offset === 0;

            return (
              <div
                key={proj.id}
                onClick={() => setCurrentIndex(idx)}
                style={{
                  transform: `perspective(850px) translateX(${x}px) translateZ(${z}px) rotateY(${rotY}deg) scale(${scale})`,
                  transformStyle: 'preserve-3d',
                  transformOrigin: offset < 0 ? 'right center' : offset > 0 ? 'left center' : 'center center',
                  zIndex,
                  opacity,
                  backgroundColor: bgColor,
                  transition: 'transform 0.45s cubic-bezier(0.25, 1, 0.5, 1), opacity 0.45s ease, background-color 0.45s ease',
                }}
                className={`absolute w-[260px] sm:w-[320px] h-[360px] sm:h-[430px] rounded-2xl sm:rounded-3xl p-6 sm:p-8 flex flex-col justify-between text-white cursor-pointer shadow-2xl overflow-hidden ${
                  isActive ? 'shadow-blue-950/40 ring-1 ring-white/20' : 'hover:brightness-110'
                }`}
              >
                {/* Dynamic 3D lighting gradient overlay (simulating depth lighting from reference) */}
                {offset < 0 && (
                  <div className="absolute inset-0 bg-gradient-to-r from-black/35 via-black/10 to-transparent pointer-events-none" />
                )}
                {offset > 0 && (
                  <div className="absolute inset-0 bg-gradient-to-l from-black/35 via-black/10 to-transparent pointer-events-none" />
                )}
                {isActive && (
                  <div className="absolute top-0 right-0 bottom-0 w-10 bg-gradient-to-l from-black/20 to-transparent pointer-events-none" />
                )}

                {/* Card Top: Category & Index */}
                <div className="flex items-center justify-between z-10">
                  <span className="text-[11px] sm:text-xs font-mono tracking-widest uppercase opacity-80 bg-black/20 px-2.5 py-1 rounded-md">
                    {proj.category}
                  </span>
                  <span className="text-xs font-mono font-bold opacity-60">
                    0{idx + 1}
                  </span>
                </div>

                {/* Card Center: Big Clean Numeral (matching the reference image) */}
                <div className="my-auto text-center z-10 flex flex-col items-center justify-center">
                  <div className="text-7xl sm:text-8xl font-bold tracking-tighter drop-shadow-sm select-none">
                    {idx + 1}
                  </div>
                </div>

                {/* Card Bottom: Project Title & Tagline */}
                <div className="z-10 mt-auto">
                  <h3 className="text-base sm:text-lg font-bold text-white tracking-tight line-clamp-1 mb-1">
                    {proj.title.split(':')[0]}
                  </h3>
                  <p className="text-xs text-blue-200/80 line-clamp-1 font-normal">
                    {proj.tagline}
                  </p>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Synchronized Project Synoptic Card */}
      {activeProject && (
        <div className="mt-8 bg-white rounded-[28px] p-8 sm:p-10 border border-zinc-200/90 shadow-sm hover:shadow-xl transition-all duration-300">
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-center">
            {/* Left: Info */}
            <div className="lg:col-span-8">
              <div className="flex flex-wrap items-center gap-2 mb-3">
                <span className="px-3 py-1 rounded-full bg-blue-50 text-blue-700 text-xs font-semibold font-mono border border-blue-200/60">
                  {activeProject.category}
                </span>
                <span className="text-xs font-mono text-zinc-400">
                  Project 0{currentIndex + 1} of 0{total}
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

          {/* Quick Select Buttons */}
          <div className="mt-8 pt-6 border-t border-zinc-100 flex items-center gap-2 overflow-x-auto pb-2">
            {projects.map((p, idx) => (
              <button
                key={p.id}
                onClick={() => setCurrentIndex(idx)}
                className={`px-3.5 py-1.5 rounded-lg text-xs font-mono transition-all shrink-0 cursor-pointer ${
                  currentIndex === idx
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
