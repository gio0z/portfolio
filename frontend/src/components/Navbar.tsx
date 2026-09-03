import React, { useState, useEffect } from 'react';
import { Menu, X, ArrowUpRight } from 'lucide-react';

interface NavbarProps {
  status?: string;
}

export const Navbar: React.FC<NavbarProps> = ({ status = 'Available for Q3/Q4 Projects' }) => {
  const [scrolled, setScrolled] = useState(false);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  useEffect(() => {
    const handleScroll = () => {
      setScrolled(window.scrollY > 20);
    };
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  const navLinks = [
    { name: 'About', href: '#about' },
    { name: 'Projects', href: '#projects' },
    { name: 'Skills', href: '#skills' },
    { name: 'Philosophy', href: '#philosophy' },
    { name: 'Contact', href: '#contact' },
  ];

  return (
    <header className={`fixed top-0 left-0 right-0 z-50 transition-all duration-300 ${
      scrolled ? 'bg-[#060913]/85 backdrop-blur-md border-b border-blue-900/30 py-3 shadow-lg shadow-blue-950/20' : 'bg-transparent py-5'
    }`}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex items-center justify-between">
        {/* Brand */}
        <a href="#" className="flex items-center gap-3 group">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-blue-700 via-blue-600 to-cyan-400 p-[1px] shadow-lg shadow-blue-600/30 group-hover:scale-105 transition-transform duration-200">
            <div className="w-full h-full bg-[#090e1f] rounded-[11px] flex items-center justify-center">
              <span className="font-bold text-lg text-transparent bg-clip-text bg-gradient-to-r from-blue-400 to-cyan-300">R</span>
            </div>
          </div>
          <div>
            <div className="font-bold text-slate-100 tracking-tight text-base sm:text-lg flex items-center gap-1.5">
              Regio Dani
              <span className="text-xs px-1.5 py-0.5 rounded bg-blue-950 text-blue-400 border border-blue-800/60 font-mono">Go+Vite</span>
            </div>
            <div className="text-xs text-blue-400/80 font-mono flex items-center gap-1.5">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
              {status}
            </div>
          </div>
        </a>

        {/* Desktop Navigation */}
        <nav className="hidden md:flex items-center gap-1 bg-slate-900/60 border border-blue-900/30 backdrop-blur-md rounded-full px-4 py-1.5 shadow-inner">
          {navLinks.map((link) => (
            <a
              key={link.name}
              href={link.href}
              className="px-4 py-1.5 text-sm font-medium text-slate-300 hover:text-white hover:bg-blue-600/15 rounded-full transition-all duration-200"
            >
              {link.name}
            </a>
          ))}
        </nav>

        {/* Action Button */}
        <div className="hidden sm:flex items-center gap-3">
          <a
            href="#contact"
            className="flex items-center gap-2 text-xs font-semibold px-4 py-2.5 rounded-xl bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-500 hover:to-blue-600 text-white shadow-lg shadow-blue-600/25 hover:shadow-blue-600/40 hover:-translate-y-0.5 transition-all duration-200"
          >
            <span>Let's Connect</span>
            <ArrowUpRight className="w-3.5 h-3.5" />
          </a>
        </div>

        {/* Mobile menu toggle */}
        <button
          onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
          className="md:hidden p-2 rounded-lg bg-blue-950/60 border border-blue-900/40 text-slate-300 hover:text-white"
        >
          {mobileMenuOpen ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
        </button>
      </div>

      {/* Mobile Menu Dropdown */}
      {mobileMenuOpen && (
        <div className="md:hidden bg-[#0a1024]/95 backdrop-blur-xl border-b border-blue-900/40 px-4 pt-3 pb-6 space-y-2 mt-2">
          {navLinks.map((link) => (
            <a
              key={link.name}
              href={link.href}
              onClick={() => setMobileMenuOpen(false)}
              className="block px-4 py-2 text-base font-medium text-slate-300 hover:text-white hover:bg-blue-600/20 rounded-lg"
            >
              {link.name}
            </a>
          ))}
          <a
            href="#contact"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-center mt-4 w-full py-2.5 text-sm font-semibold rounded-lg bg-blue-600 text-white"
          >
            Let's Connect
          </a>
        </div>
      )}
    </header>
  );
};
