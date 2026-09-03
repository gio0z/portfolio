import React from 'react';
import { ArrowRight, Send, Terminal, Sparkles, ShieldCheck, Zap } from 'lucide-react';
import type { Profile } from '../types';

interface HeroProps {
  profile: Profile | null;
}

export const Hero: React.FC<HeroProps> = ({ profile }) => {
  return (
    <section id="about" className="relative pt-32 pb-20 md:pt-40 md:pb-28 overflow-hidden">
      {/* Background Decorative Ambient Glows */}
      <div className="absolute top-1/4 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[350px] bg-blue-600/20 blur-[140px] rounded-full pointer-events-none -z-10" />
      <div className="absolute top-20 right-10 w-[300px] h-[300px] bg-cyan-500/10 blur-[100px] rounded-full pointer-events-none -z-10" />
      <div className="absolute top-40 left-10 w-[250px] h-[250px] bg-indigo-600/15 blur-[90px] rounded-full pointer-events-none -z-10" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex flex-col items-center text-center max-w-4xl mx-auto">
          {/* Top Pill / Status */}
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bg-blue-950/70 border border-blue-800/40 text-blue-300 text-xs sm:text-sm font-medium mb-8 shadow-inner shadow-blue-500/10 backdrop-blur-md">
            <span className="flex h-2 w-2 relative">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-cyan-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2 w-2 bg-cyan-500"></span>
            </span>
            <span className="text-slate-300 font-mono tracking-wide">ARCHITECTURE • DISTRIBUTED SYSTEMS • AI AGENTS</span>
          </div>

          {/* Main Headline */}
          <h1 className="text-4xl sm:text-6xl lg:text-7xl font-extrabold tracking-tight text-white mb-6 leading-[1.15]">
            Engineering Resilient <br className="hidden sm:inline" />
            <span className="bg-gradient-to-r from-blue-400 via-sky-300 to-cyan-300 bg-clip-text text-transparent">
              Backends & Autonomous Agents
            </span>
          </h1>

          {/* Subtitle */}
          <p className="text-base sm:text-xl text-slate-300 mb-10 max-w-2xl font-normal leading-relaxed">
            I'm <span className="text-white font-semibold">{profile?.name || 'Regio Dani Pangestu'}</span>. 
            I architect high-concurrency Go microservices, lightning-fast reactive web frontends with Vite, 
            and self-improving multi-agent workflows built on disciplined engineering standards.
          </p>

          {/* CTAs */}
          <div className="flex flex-wrap items-center justify-center gap-4 mb-16 w-full sm:w-auto">
            <a
              href="#projects"
              className="flex items-center justify-center gap-2.5 px-6 py-3.5 rounded-xl bg-gradient-to-r from-blue-600 via-blue-700 to-indigo-700 hover:from-blue-500 hover:to-blue-600 text-white font-semibold text-sm shadow-xl shadow-blue-700/30 hover:shadow-blue-600/50 hover:-translate-y-0.5 transition-all duration-200 w-full sm:w-auto"
            >
              <span>Explore Selected Work</span>
              <ArrowRight className="w-4 h-4" />
            </a>

            <a
              href="https://github.com/gio0z"
              target="_blank"
              rel="noreferrer"
              className="flex items-center justify-center gap-2.5 px-6 py-3.5 rounded-xl bg-slate-900/80 hover:bg-slate-800/80 border border-blue-900/40 hover:border-blue-700/60 text-slate-200 hover:text-white font-semibold text-sm backdrop-blur-md transition-all duration-200 w-full sm:w-auto shadow-md"
            >
              <svg className="w-4 h-4 fill-current" viewBox="0 0 24 24">
                <path fillRule="evenodd" clipRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" />
              </svg>
              <span>GitHub /gio0z</span>
            </a>

            <a
              href="#contact"
              className="flex items-center justify-center gap-2 px-5 py-3.5 rounded-xl bg-blue-950/40 hover:bg-blue-900/40 border border-blue-800/30 text-blue-300 hover:text-blue-200 font-semibold text-sm transition-all duration-200 w-full sm:w-auto"
            >
              <Send className="w-4 h-4" />
              <span>Contact Direct</span>
            </a>
          </div>

          {/* Quick Technical Badges */}
          <div className="flex flex-wrap justify-center items-center gap-2.5 text-xs text-slate-400 font-mono mb-16">
            <span className="text-slate-500">Core Stack:</span>
            {['Go (Golang)', 'Vite', 'React 19', 'TypeScript', 'Tailwind v4', 'Hermes Agent', 'Docker', 'PostgreSQL'].map((tech) => (
              <span key={tech} className="px-2.5 py-1 rounded-md bg-slate-900/70 border border-blue-900/30 text-blue-300">
                {tech}
              </span>
            ))}
          </div>
        </div>

        {/* Stats Grid Bar (Glass Cards) */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 max-w-5xl mx-auto">
          {profile?.stats ? (
            profile.stats.map((stat, idx) => (
              <div
                key={stat.label}
                className="p-5 rounded-2xl bg-[#0b1329]/70 border border-blue-900/30 backdrop-blur-md hover:border-blue-500/40 hover:shadow-lg hover:shadow-blue-900/20 transition-all duration-300 group"
              >
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs font-mono uppercase tracking-wider text-slate-400 group-hover:text-blue-400 transition-colors">
                    {stat.label}
                  </span>
                  {idx === 0 && <Terminal className="w-4 h-4 text-blue-400" />}
                  {idx === 1 && <Zap className="w-4 h-4 text-cyan-400" />}
                  {idx === 2 && <ShieldCheck className="w-4 h-4 text-emerald-400" />}
                  {idx === 3 && <Sparkles className="w-4 h-4 text-sky-400" />}
                </div>
                <div className="text-2xl sm:text-3xl font-extrabold text-white tracking-tight mb-1 group-hover:text-blue-200 transition-colors">
                  {stat.value}
                </div>
                <div className="text-xs text-slate-400 truncate">
                  {stat.sub}
                </div>
              </div>
            ))
          ) : (
            <div className="col-span-4 text-center py-4 text-slate-500 text-sm">Loading telemetry...</div>
          )}
        </div>
      </div>
    </section>
  );
};
