import React from 'react';
import { ArrowUp, Terminal } from 'lucide-react';

export const Footer: React.FC = () => {
  const scrollToTop = () => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  return (
    <footer className="py-12 border-t border-blue-900/30 bg-[#050813] text-slate-400 text-xs">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex flex-col sm:flex-row items-center justify-between gap-6">
        <div className="flex items-center gap-3">
          <div className="w-7 h-7 rounded-lg bg-blue-950 border border-blue-800/60 flex items-center justify-center text-blue-400 font-bold font-mono">
            R
          </div>
          <div>
            <span className="font-semibold text-slate-200">Regio Dani Pangestu</span>
            <span className="text-slate-500 mx-2">•</span>
            <span>Architected with Go & Vite</span>
          </div>
        </div>

        <div className="flex items-center gap-6">
          <div className="flex items-center gap-2 text-slate-400 font-mono text-[11px]">
            <Terminal className="w-3.5 h-3.5 text-blue-400" />
            <span>TDD Seams: Verified PASS</span>
          </div>

          <button
            onClick={scrollToTop}
            aria-label="Scroll to top"
            className="p-2.5 rounded-lg bg-slate-900 hover:bg-slate-800 border border-blue-900/30 text-slate-300 hover:text-white transition-all cursor-pointer"
          >
            <ArrowUp className="w-4 h-4" />
          </button>
        </div>
      </div>
    </footer>
  );
};
