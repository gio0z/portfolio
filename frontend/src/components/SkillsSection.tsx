import React from 'react';
import { Cpu, Shield, Database, Network, Zap, Code2, Palette, Layout, Bot, Sparkles, Layers, Brain, Container, Terminal, GitBranch } from 'lucide-react';
import type { SkillCategory } from '../types';

interface SkillsSectionProps {
  categories: SkillCategory[];
}

export const SkillsSection: React.FC<SkillsSectionProps> = ({ categories }) => {
  const getIcon = (iconName: string) => {
    switch (iconName) {
      case 'Cpu': return <Cpu className="w-4 h-4 text-blue-400" />;
      case 'Shield': return <Shield className="w-4 h-4 text-cyan-400" />;
      case 'Database': return <Database className="w-4 h-4 text-indigo-400" />;
      case 'Network': return <Network className="w-4 h-4 text-sky-400" />;
      case 'Zap': return <Zap className="w-4 h-4 text-amber-400" />;
      case 'Code2': return <Code2 className="w-4 h-4 text-blue-400" />;
      case 'Palette': return <Palette className="w-4 h-4 text-purple-400" />;
      case 'Layout': return <Layout className="w-4 h-4 text-teal-400" />;
      case 'Bot': return <Bot className="w-4 h-4 text-cyan-400" />;
      case 'Sparkles': return <Sparkles className="w-4 h-4 text-yellow-400" />;
      case 'Layers': return <Layers className="w-4 h-4 text-blue-400" />;
      case 'Brain': return <Brain className="w-4 h-4 text-pink-400" />;
      case 'Container': return <Container className="w-4 h-4 text-sky-400" />;
      case 'Terminal': return <Terminal className="w-4 h-4 text-emerald-400" />;
      case 'GitBranch': return <GitBranch className="w-4 h-4 text-orange-400" />;
      default: return <Cpu className="w-4 h-4 text-blue-400" />;
    }
  };

  return (
    <section id="skills" className="py-24 relative bg-gradient-to-b from-transparent via-[#080f22]/50 to-transparent">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center max-w-3xl mx-auto mb-16">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-blue-950/80 border border-blue-800/40 text-blue-400 text-xs font-mono uppercase tracking-wider mb-4">
            <Cpu className="w-3.5 h-3.5" />
            <span>Capability Matrix</span>
          </div>
          <h2 className="text-3xl sm:text-5xl font-extrabold text-white tracking-tight">
            Technical Architecture & Tooling
          </h2>
          <p className="text-slate-400 text-sm sm:text-base mt-3">
            Core mastery of high-throughput backend infrastructure, reactive web client stacks, and autonomous AI agents.
          </p>
        </div>

        {/* Categories Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
          {categories.map((cat) => (
            <div
              key={cat.category}
              className="p-6 sm:p-8 rounded-2xl bg-[#091124]/80 border border-blue-900/30 backdrop-blur-md hover:border-blue-700/50 hover:shadow-xl hover:shadow-blue-950/40 transition-all duration-300"
            >
              <div className="mb-6">
                <h3 className="text-xl font-bold text-white tracking-tight flex items-center gap-2.5">
                  <span className="w-2 h-2 rounded-full bg-blue-500"></span>
                  {cat.category}
                </h3>
                <p className="text-xs text-slate-400 mt-1">
                  {cat.summary}
                </p>
              </div>

              <div className="space-y-5">
                {cat.skills.map((skill) => (
                  <div key={skill.name} className="space-y-1.5">
                    <div className="flex items-center justify-between text-xs sm:text-sm">
                      <div className="flex items-center gap-2 font-medium text-slate-200">
                        {getIcon(skill.icon)}
                        <span>{skill.name}</span>
                      </div>
                      <div className="flex items-center gap-2 font-mono text-xs">
                        <span className="text-blue-400 font-semibold">{skill.proficiency}</span>
                        <span className="text-slate-500">{skill.level}%</span>
                      </div>
                    </div>

                    {/* Progress Bar with Blue Gradient */}
                    <div className="w-full h-2 rounded-full bg-slate-900/90 overflow-hidden border border-slate-800/80">
                      <div
                        className="h-full rounded-full bg-gradient-to-r from-blue-600 via-blue-500 to-cyan-400 transition-all duration-1000"
                        style={{ width: `${skill.level}%` }}
                      />
                    </div>

                    <p className="text-[11px] text-slate-400/90 font-normal">
                      {skill.description}
                    </p>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};
