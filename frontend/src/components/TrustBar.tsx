import React from 'react';
import { Hexagon, Sparkles, Cpu, Layers, Disc3 } from 'lucide-react';

export const TrustBar: React.FC = () => {
  const clients = [
    { name: 'CozyNest', icon: <Hexagon className="w-5 h-5 text-zinc-400" /> },
    { name: 'LuxeAura', icon: <Sparkles className="w-5 h-5 text-zinc-400" /> },
    { name: 'ZestyBite', icon: <Cpu className="w-5 h-5 text-zinc-400" /> },
    { name: 'DigiMinds', icon: <Layers className="w-5 h-5 text-zinc-400" /> },
    { name: 'Ensphere', icon: <Disc3 className="w-5 h-5 text-zinc-400" /> },
  ];

  return (
    <section className="py-12 border-y border-zinc-200/80 bg-white/60">
      <div className="max-w-7xl mx-auto px-4 sm:px-8">
        <div className="flex flex-wrap items-center justify-between gap-8 sm:gap-12 opacity-70 grayscale hover:grayscale-0 transition-all duration-300">
          {clients.map((c) => (
            <div key={c.name} className="flex items-center gap-2.5 text-zinc-600 font-bold text-base sm:text-lg tracking-tight">
              {c.icon}
              <span>{c.name}</span>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};
