import React from 'react';
import { ArrowUp } from 'lucide-react';

export const Footer: React.FC = () => {
  const scrollToTop = () => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  return (
    <footer className="py-12 px-4 sm:px-8 max-w-7xl mx-auto border-t border-zinc-200 text-zinc-500 text-xs sm:text-sm">
      <div className="flex flex-col sm:flex-row items-center justify-between gap-6">
        <div className="flex items-center gap-2">
          <span className="font-extrabold text-base text-blue-600 tracking-tight">regio.</span>
          <span>© {new Date().getFullYear()} Regio Dani Pangestu. All rights reserved.</span>
        </div>

        <div className="flex items-center gap-6">
          <span className="font-mono text-xs text-zinc-400">Architecture: Go + Vite</span>
          <button
            onClick={scrollToTop}
            aria-label="Back to top"
            className="w-9 h-9 rounded-full bg-zinc-200 hover:bg-zinc-300 flex items-center justify-center text-zinc-800 transition-colors cursor-pointer"
          >
            <ArrowUp className="w-4 h-4" />
          </button>
        </div>
      </div>
    </footer>
  );
};
