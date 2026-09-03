import React, { useState } from 'react';
import { ArrowDownRight, CheckCircle2, ArrowRight } from 'lucide-react';

export const ExpertiseSection: React.FC = () => {
  const [activeTab, setActiveTab] = useState(0);

  const tabs = [
    {
      id: '01',
      title: 'Go & Distributed Systems',
      short: 'Go Systems',
      tag: 'BACKEND ARCHITECTURE',
      description: 'Ultra-low latency microservices with Go net/http, gRPC, and strict concurrency safety. Optimized for high-throughput transactional integrity and sub-50ms response windows.',
      capabilities: ['Goroutines & Channel Pipelines', 'PostgreSQL / SQLite Storage Engine', 'Idempotent REST & gRPC Contracts', 'Zero-Downtime Releases'],
    },
    {
      id: '02',
      title: 'Vite & Reactive Web Platforms',
      short: 'Vite & React',
      tag: 'FRONTEND ARCHITECTURE',
      description: 'High-speed SPA and SSR frontends built with Vite, React 19, TypeScript, and modern Tailwind CSS design systems. Engineered for instant HMR and sub-second bundle performance.',
      capabilities: ['Sub-second Rollup Bundles', 'Type-Safe API Client Integrations', 'Swiss & Pinterest Minimalist Aesthetics', 'Fluid Responsive UI Architecture'],
    },
    {
      id: '03',
      title: 'Autonomous AI Agents & Mesh',
      short: 'AI Agents',
      tag: 'AGENTIC COMPUTING',
      description: 'Multi-platform autonomous agent mesh orchestrating Hermes Agent, Open Design, and MCP tools across WhatsApp Baileys, Telegram, and enterprise platforms.',
      capabilities: ['Multi-Platform Gateway Routing', 'Hindsight Long-Term Memory', 'Custom MCP Server Authoring', 'Subagent Parallel Workflows'],
    },
  ];

  return (
    <section id="expertise" className="py-24 sm:py-32 px-4 sm:px-8 max-w-7xl mx-auto">
      {/* Asymmetric Header */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-end mb-16 sm:mb-20">
        <div className="lg:col-span-7">
          <div className="inline-flex items-center gap-2 text-xs font-semibold tracking-wider uppercase text-zinc-500 mb-5 font-mono">
            <span className="flex items-center justify-center w-4 h-4 rounded bg-blue-600 text-white">
              <ArrowDownRight className="w-3 h-3" />
            </span>
            <span>Expertise</span>
          </div>

          <h2 className="text-4xl sm:text-5xl lg:text-6xl font-extrabold tracking-tight text-zinc-900 leading-[1.1]">
            <span>Growth-Focused</span> <br />
            <span className="text-zinc-400 font-bold">Systems Solutions</span>
          </h2>
        </div>

        <div className="lg:col-span-5">
          <p className="text-base sm:text-lg text-zinc-600 leading-relaxed">
            From Performance-Focused Architecture To Technical Implementation, I Provide The Resilient Infrastructure and Agentic Tools You Need To Dominate Your Industry.
          </p>
        </div>
      </div>

      {/* Dark Footer Bar / Drawer Strip */}
      <div className="bg-[#18181B] text-white rounded-[32px] overflow-hidden border border-white/10 shadow-2xl">
        {/* Top Tabs Bar */}
        <div className="grid grid-cols-1 md:grid-cols-3 divide-y md:divide-y-0 md:divide-x divide-zinc-800">
          {tabs.map((tab, idx) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(idx)}
              className={`p-6 sm:p-8 text-left transition-all duration-200 cursor-pointer group ${
                activeTab === idx ? 'bg-[#222226]' : 'hover:bg-[#1E1E22]'
              }`}
            >
              <div className="flex items-center justify-between text-xs font-mono text-zinc-400 mb-3">
                <span className={activeTab === idx ? 'text-blue-400 font-bold' : ''}>
                  {tab.id} — {tab.short}
                </span>
                <ArrowRight className={`w-4 h-4 transition-transform ${
                  activeTab === idx ? 'translate-x-1 text-blue-400' : 'text-zinc-600 group-hover:translate-x-0.5'
                }`} />
              </div>
              <div className="text-2xl sm:text-3xl font-extrabold text-white tracking-tight">
                {tab.short}
              </div>
            </button>
          ))}
        </div>

        {/* Selected Drawer Detail Panel */}
        <div className="p-8 sm:p-12 bg-[#1C1C20] border-t border-zinc-800">
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-center">
            <div className="lg:col-span-7">
              <span className="text-xs font-mono text-blue-400 tracking-wider uppercase font-semibold">
                {tabs[activeTab].tag}
              </span>
              <h3 className="text-2xl sm:text-3xl font-bold text-white mt-1 mb-4">
                {tabs[activeTab].title}
              </h3>
              <p className="text-zinc-400 text-sm sm:text-base leading-relaxed max-w-2xl">
                {tabs[activeTab].description}
              </p>
            </div>

            <div className="lg:col-span-5 bg-[#25252A] rounded-2xl p-6 border border-white/5">
              <div className="text-xs font-mono uppercase text-zinc-400 mb-4 tracking-wider">
                Key Architectural Capabilities
              </div>
              <div className="space-y-3">
                {tabs[activeTab].capabilities.map((cap) => (
                  <div key={cap} className="flex items-center gap-3 text-sm text-zinc-200 font-medium">
                    <CheckCircle2 className="w-4 h-4 text-blue-500 shrink-0" />
                    <span>{cap}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};
