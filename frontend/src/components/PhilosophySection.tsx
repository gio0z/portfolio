import React from 'react';
import { CheckCircle2, ShieldAlert, GitCommit, Sparkles } from 'lucide-react';

export const PhilosophySection: React.FC = () => {
  const principles = [
    {
      title: 'Disciplined Test-Driven Development (TDD)',
      tag: 'RED-GREEN-REFACTOR',
      description:
        'Tests are written against public seams before any code is generated. No tautological tests, no mock pollution. Every slice starts red, turns green with minimal code, and remains a durable behavioral specification.',
      icon: <CheckCircle2 className="w-5 h-5 text-cyan-400" />,
      color: 'from-blue-600/20 to-cyan-500/10',
    },
    {
      title: 'Deep Modules & Minimal Surface Area',
      tag: 'Ousterhout Philosophy',
      description:
        'Modules should be deep: maximizing leverage by packing substantial complex functionality behind clean, simple, and intuitive interfaces. High cohesion, low coupling, and clear seam boundaries.',
      icon: <ShieldAlert className="w-5 h-5 text-blue-400" />,
      color: 'from-blue-600/20 to-indigo-500/10',
    },
    {
      title: 'Evidence-Based Engineering',
      tag: 'VERIFICATION BEFORE ASSERTION',
      description:
        'Never substitute plausible-looking assertions for real execution. Every change, build, and API response is backed by deterministic tool output, real test runs, and observable telemetry.',
      icon: <GitCommit className="w-5 h-5 text-sky-400" />,
      color: 'from-sky-600/20 to-blue-500/10',
    },
    {
      title: 'Autonomous Multi-Agent Architecture',
      tag: 'HERMES & SUPERPOWERS',
      description:
        'Leveraging AI agents not for fragile autocomplete, but as disciplined, subagent-driven engineering collaborators capable of parallel planning, rigorous code reviews, and autonomous execution.',
      icon: <Sparkles className="w-5 h-5 text-amber-400" />,
      color: 'from-indigo-600/20 to-cyan-500/10',
    },
  ];

  return (
    <section id="philosophy" className="py-24 relative overflow-hidden">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="max-w-3xl mb-16">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-blue-950/80 border border-blue-800/40 text-blue-400 text-xs font-mono uppercase tracking-wider mb-4">
            <Sparkles className="w-3.5 h-3.5" />
            <span>Engineering Discipline</span>
          </div>
          <h2 className="text-3xl sm:text-5xl font-extrabold text-white tracking-tight">
            How I Build Software
          </h2>
          <p className="text-slate-400 text-sm sm:text-base mt-3">
            Core tenets derived from decades of software architecture wisdom and modern agentic engineering.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {principles.map((p) => (
            <div
              key={p.title}
              className="p-7 rounded-2xl bg-[#0a1329]/70 border border-blue-900/30 hover:border-blue-500/40 hover:shadow-xl hover:shadow-blue-900/15 backdrop-blur-md transition-all duration-300"
            >
              <div className="flex items-start justify-between mb-4">
                <div className="w-10 h-10 rounded-xl bg-blue-950/90 border border-blue-800/60 flex items-center justify-center">
                  {p.icon}
                </div>
                <span className="text-[11px] font-mono uppercase tracking-wider px-2.5 py-1 rounded bg-blue-950/90 text-cyan-300 border border-blue-900/50">
                  {p.tag}
                </span>
              </div>
              <h3 className="text-lg font-bold text-white mb-2 tracking-tight">
                {p.title}
              </h3>
              <p className="text-sm text-slate-300 leading-relaxed font-normal">
                {p.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};
