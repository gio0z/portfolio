import React from 'react';
import { ArrowDownRight, Search, GitFork, BarChart3 } from 'lucide-react';

export const PhilosophySection: React.FC = () => {
  const pillars = [
    {
      title: 'System Audits',
      icon: <Search className="w-8 h-8 text-zinc-400 stroke-[1.75]" />,
      description: 'Identify gaps and bottlenecks in your current backend architecture to optimize concurrency, throughput, and system reliability.',
    },
    {
      title: 'High-Concurrency Engines',
      icon: <GitFork className="w-8 h-8 text-zinc-400 stroke-[1.75]" />,
      description: 'Build scalable Go microservices and automated event pipelines that handle high-volume distributed requests with zero jitter.',
    },
    {
      title: 'Data-Driven ROI & TDD',
      icon: <BarChart3 className="w-8 h-8 text-zinc-400 stroke-[1.75]" />,
      description: 'Transform complex domain logic into disciplined red-green tests and measurable business impact for faster, deterministic releases.',
    },
  ];

  return (
    <section id="philosophy" className="py-24 sm:py-32 px-4 sm:px-8 max-w-7xl mx-auto">
      {/* Header */}
      <div className="text-center max-w-3xl mx-auto mb-16 sm:mb-20">
        <div className="inline-flex items-center gap-2 text-xs font-semibold tracking-wider uppercase text-zinc-500 mb-5 font-mono">
          <span className="flex items-center justify-center w-4 h-4 rounded bg-blue-600 text-white">
            <ArrowDownRight className="w-3 h-3" />
          </span>
          <span>My Philosophy</span>
        </div>

        <h2 className="text-4xl sm:text-5xl lg:text-6xl font-extrabold tracking-tight text-zinc-900 leading-[1.1] mb-6">
          <span>Architect-Led</span> <br />
          <span className="text-zinc-400 font-bold">Built On Data & TDD</span>
        </h2>

        <p className="text-base sm:text-lg text-zinc-600 leading-relaxed max-w-2xl mx-auto">
          I Don't Just Write Code; I Build Resilient Systems Backed By Rigorous Analysis and Verification To Ensure Your Objectives Are Met With Precision.
        </p>
      </div>

      {/* 3-Column Pillar Cards Row */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 sm:gap-8">
        {pillars.map((pillar) => (
          <div
            key={pillar.title}
            className="bg-white rounded-[28px] p-8 sm:p-10 border border-zinc-200/90 shadow-sm hover:shadow-xl hover:border-blue-500/30 hover:-translate-y-1 transition-all duration-300 flex flex-col justify-between"
          >
            <div>
              <h3 className="text-xl sm:text-2xl font-bold text-zinc-900 mb-8 tracking-tight">
                {pillar.title}
              </h3>

              {/* Centered Large Monochromatic Icon */}
              <div className="w-20 h-20 rounded-full bg-zinc-100 flex items-center justify-center mx-auto my-6 border border-zinc-200/70">
                {pillar.icon}
              </div>
            </div>

            <p className="text-sm sm:text-base text-zinc-600 leading-relaxed text-center mt-6">
              {pillar.description}
            </p>
          </div>
        ))}
      </div>
    </section>
  );
};
