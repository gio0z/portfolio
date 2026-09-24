import React from 'react';
import { clients } from '../data/clients';

export const TrustBar: React.FC = () => {
  return (
    <section className="py-12 border-y border-zinc-200/80 bg-white/60">
      <div className="max-w-7xl mx-auto px-4 sm:px-8">
        <div className="flex flex-wrap items-center justify-between gap-8 sm:gap-12 opacity-70 grayscale hover:grayscale-0 transition-all duration-300">
          {clients.map((c) => (
            <div key={c.name} className="flex items-center gap-2.5 text-zinc-600 font-bold text-base sm:text-lg tracking-tight">
              <span>{c.name}</span>
              <span className="text-xs text-zinc-400 font-mono">{c.sector}</span>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};
