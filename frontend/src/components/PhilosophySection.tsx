import React from 'react';
import { ArrowDownRight, Search, GitFork, BarChart3 } from 'lucide-react';

export const PhilosophySection: React.FC = () => {
  const pillars = [
    {
      title: 'Look before building',
      icon: <Search className="w-8 h-8 text-zinc-400 stroke-[1.75]" />,
      description: 'I map what already exists and where it breaks, so the fix targets the real bottleneck instead of the loudest one.',
    },
    {
      title: 'Build for the busy day',
      icon: <GitFork className="w-8 h-8 text-zinc-400 stroke-[1.75]" />,
      description: 'Design for your peak, not your average. The system should stay correct when traffic is ten times normal.',
    },
    {
      title: 'Prove it works',
      icon: <BarChart3 className="w-8 h-8 text-zinc-400 stroke-[1.75]" />,
      description: 'Every change lands with tests that fail without it, so progress is measured rather than assumed.',
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
          <span>How I Work</span>
        </div>

        <h2 className="text-4xl sm:text-5xl lg:text-6xl font-extrabold tracking-tight text-zinc-900 leading-[1.1] mb-6">
          <span>Built to last,</span> <br />
          <span className="text-zinc-400 font-bold">not just to ship.</span>
        </h2>

        <p className="text-base sm:text-lg text-zinc-600 leading-relaxed max-w-2xl mx-auto">
          Anyone can make software work once. The job is making it keep working — after the launch, after the next feature, after you hand it to someone else.
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
