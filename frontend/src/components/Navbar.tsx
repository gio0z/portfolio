import React, { useState } from 'react';
import { Menu, X, ArrowDownRight } from 'lucide-react';

export const Navbar: React.FC = () => {
  const [mobileOpen, setMobileOpen] = useState(false);

  const links = [
    { name: 'Home', href: '#' },
    { name: 'About', href: '#about' },
    { name: 'Projects', href: '#projects' },
    { name: 'Philosophy', href: '#philosophy' },
    { name: 'Expertise', href: '#expertise' },
    { name: 'Contact', href: '#contact' },
  ];

  return (
    <header className="sticky top-4 z-50 px-4 sm:px-8 max-w-7xl mx-auto">
      <div className="bg-[#18181B] text-white rounded-2xl sm:rounded-full px-5 sm:px-7 py-3.5 flex items-center justify-between shadow-2xl shadow-black/15 border border-white/5">
        {/* Left: Brand Logo */}
        <a href="#" className="flex items-center gap-1 group">
          <span className="font-extrabold text-xl tracking-tight text-blue-500 group-hover:text-blue-400 transition-colors">
            regio<span className="text-white">.</span>
          </span>
        </a>

        {/* Center: Desktop Navigation */}
        <nav className="hidden md:flex items-center gap-8">
          {links.map((link) => (
            <a
              key={link.name}
              href={link.href}
              className="text-sm font-medium text-zinc-300 hover:text-white transition-colors duration-150"
            >
              {link.name}
            </a>
          ))}
        </nav>

        {/* Right: Actions */}
        <div className="hidden sm:flex items-center gap-4">
          <div className="flex items-center gap-1.5 text-xs text-zinc-400 font-mono">
            <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
            <span>Active</span>
          </div>

          <a
            href="#contact"
            className="flex items-center gap-1.5 px-5 py-2.5 rounded-full bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs transition-all duration-200 shadow-md shadow-blue-600/30 hover:scale-[1.02]"
          >
            <span>Book A Call</span>
            <ArrowDownRight className="w-3.5 h-3.5" />
          </a>
        </div>

        {/* Mobile menu button */}
        <button
          onClick={() => setMobileOpen(!mobileOpen)}
          className="md:hidden p-1.5 text-zinc-300 hover:text-white rounded-lg focus:outline-none"
        >
          {mobileOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
        </button>
      </div>

      {/* Mobile Dropdown */}
      {mobileOpen && (
        <div className="md:hidden mt-2 bg-[#18181B] text-white rounded-2xl p-5 border border-white/10 shadow-2xl space-y-3">
          {links.map((link) => (
            <a
              key={link.name}
              href={link.href}
              onClick={() => setMobileOpen(false)}
              className="block text-base font-medium text-zinc-300 hover:text-white py-1"
            >
              {link.name}
            </a>
          ))}
          <a
            href="#contact"
            onClick={() => setMobileOpen(false)}
            className="flex items-center justify-center gap-2 w-full py-3 rounded-xl bg-blue-600 text-white text-sm font-semibold mt-2"
          >
            <span>Book A Call</span>
            <ArrowDownRight className="w-4 h-4" />
          </a>
        </div>
      )}
    </header>
  );
};
