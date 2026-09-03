import React from 'react';
import { ArrowDownRight } from 'lucide-react';
import type { Profile } from '../types';

interface HeroProps {
  profile: Profile | null;
}

export const Hero: React.FC<HeroProps> = ({ profile }) => {
  return (
    <section className="pt-12 sm:pt-20 pb-16 sm:pb-24 px-4 sm:px-8 max-w-7xl mx-auto">
      {/* 1. Hero Text Header */}
      <div className="text-center max-w-4xl mx-auto mb-16 sm:mb-20">
        {/* Pre-title Eyebrow */}
        <div className="inline-flex items-center gap-2 text-xs sm:text-sm font-semibold tracking-wider uppercase text-zinc-500 mb-6 font-mono">
          <span className="flex items-center justify-center w-5 h-5 rounded-md bg-blue-600 text-white">
            <ArrowDownRight className="w-3.5 h-3.5" />
          </span>
          <span>Welcome to Regio</span>
        </div>

        {/* Monumental 3-Line Headline */}
        <h1 className="text-4xl sm:text-6xl lg:text-7xl font-extrabold tracking-tight text-zinc-900 leading-[1.08] mb-6">
          <span>Your Next Best</span> <br />
          <span className="inline-flex items-center gap-3">
            <span>Engineering</span>
            <span className="inline-flex items-center justify-center w-9 h-9 sm:w-14 sm:h-14 rounded-xl sm:rounded-2xl bg-blue-600 text-white shadow-lg shadow-blue-600/30">
              <ArrowDownRight className="w-5 h-5 sm:w-8 sm:h-8" />
            </span>
            <span className="text-zinc-400 font-bold">Decision</span>
          </span> <br />
          <span className="text-zinc-400 font-bold">Starts Here</span>
        </h1>

        {/* Supporting Paragraph */}
        <p className="text-base sm:text-lg text-zinc-600 max-w-2xl mx-auto leading-relaxed">
          I'm a full-stack engineer and AI systems architect helping businesses grow through resilient Go backends, reactive Vite frontends, and autonomous multi-agent pipelines. Browse my work — the numbers do the talking.
        </p>
      </div>

      {/* 2. Split Two-Column Feature Cards (50/50 Bento Grid) */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 lg:gap-8 items-stretch">
        {/* Left Column: Dark Pitch Card */}
        <div className="lg:col-span-6 bg-[#18181B] text-white rounded-[32px] p-8 sm:p-12 flex flex-col justify-between shadow-xl relative overflow-hidden border border-white/5">
          {/* Top Tag & Header */}
          <div>
            <div className="inline-flex items-center gap-2 text-xs font-semibold tracking-wide uppercase text-zinc-400 mb-6 font-mono">
              <span className="flex items-center justify-center w-4 h-4 rounded bg-blue-600 text-white">
                <ArrowDownRight className="w-3 h-3" />
              </span>
              <span>About the engineer</span>
            </div>

            <h2 className="text-3xl sm:text-4xl lg:text-5xl font-extrabold tracking-tight leading-[1.1] mb-6">
              <span>Systems first.</span> <br />
              <span className="text-zinc-500 font-bold">Always.</span>
            </h2>

            <p className="text-zinc-400 text-sm sm:text-base leading-relaxed mb-8 max-w-xl">
              I lead projects of high-concurrency distributed backends and reactive web frontends from a clear foundation, combining strict performance criteria and visual sensitivity to build systems coherent and durable.
            </p>

            {/* Action Buttons */}
            <div className="flex flex-wrap items-center gap-3 mb-10">
              <a
                href="#about"
                className="flex items-center gap-2 px-6 py-3 rounded-full bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs sm:text-sm shadow-md shadow-blue-600/30 transition-all duration-200"
              >
                <span>More About Me</span>
                <ArrowDownRight className="w-4 h-4" />
              </a>

              <a
                href="#projects"
                className="flex items-center gap-2 px-6 py-3 rounded-full bg-white hover:bg-zinc-100 text-zinc-900 font-semibold text-xs sm:text-sm shadow-md transition-all duration-200"
              >
                <span>See Projects</span>
                <ArrowDownRight className="w-4 h-4" />
              </a>
            </div>
          </div>

          {/* Embedded Highlight Card (Bottom) */}
          <div className="bg-[#242428] rounded-2xl p-4 sm:p-5 flex items-center gap-4 border border-white/10 hover:border-blue-500/40 transition-colors group">
            <div className="w-16 h-16 sm:w-20 sm:h-20 rounded-xl overflow-hidden shrink-0 bg-zinc-800">
              <img
                src="https://images.unsplash.com/photo-1558494949-ef010cbdcc31?auto=format&fit=crop&w=300&q=80"
                alt="Highlight case study"
                className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
              />
            </div>
            <div className="flex-1 min-w-0">
              <div className="text-xs font-semibold text-white group-hover:text-blue-400 transition-colors line-clamp-2 leading-snug mb-1">
                Increasing Microservice Concurrency from 10K to 50K RPS
              </div>
              <a
                href="#projects"
                className="inline-flex items-center gap-1 text-xs font-semibold text-blue-400 hover:text-blue-300 transition-colors"
              >
                <span>See Details</span>
                <ArrowDownRight className="w-3.5 h-3.5" />
              </a>
            </div>
          </div>
        </div>

        {/* Right Column: Photograph & Floating Metric Overlays */}
        <div className="lg:col-span-6 relative rounded-[32px] overflow-hidden min-h-[500px] sm:min-h-[600px] flex flex-col justify-end p-4 sm:p-6 shadow-xl bg-zinc-200">
          {/* Strategist / Engineer Photo */}
          <img
            src="https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?auto=format&fit=crop&w=1200&q=80"
            alt={profile?.name || 'Regio Dani Pangestu'}
            className="absolute inset-0 w-full h-full object-cover object-top"
          />

          {/* Subtle bottom shadow overlay to guarantee metric card readability */}
          <div className="absolute inset-0 bg-gradient-to-t from-black/40 via-transparent to-transparent pointer-events-none" />

          {/* Floating Metric Badges Cluster (2 Cards Side-by-Side) */}
          <div className="relative z-10 grid grid-cols-1 sm:grid-cols-2 gap-3 w-full">
            {/* Metric 1 */}
            <div className="bg-white/95 backdrop-blur-md rounded-2xl p-4 sm:p-5 shadow-2xl border border-black/5">
              <div className="text-3xl sm:text-4xl font-extrabold text-zinc-900 tracking-tight mb-1">
                3.8<span className="text-blue-600 font-bold">x</span>
              </div>
              <div className="text-xs font-medium text-zinc-600 mb-2 leading-tight">
                Throughput efficiency across all Go microservices
              </div>
              <div className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-600 bg-emerald-50 px-2 py-0.5 rounded-md font-mono">
                <span>↑ 64% above industry avg</span>
              </div>
            </div>

            {/* Metric 2 */}
            <div className="bg-white/95 backdrop-blur-md rounded-2xl p-4 sm:p-5 shadow-2xl border border-black/5">
              <div className="text-3xl sm:text-4xl font-extrabold text-zinc-900 tracking-tight mb-1">
                +240<span className="text-blue-600 font-bold">%</span>
              </div>
              <div className="text-xs font-medium text-zinc-600 mb-2 leading-tight">
                Velocity gain via TDD · average 6 months
              </div>
              <div className="inline-flex items-center gap-1 text-[11px] font-semibold text-blue-600 bg-blue-50 px-2 py-0.5 rounded-md font-mono">
                <span>↑ Sub-50ms p99 latency</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};
