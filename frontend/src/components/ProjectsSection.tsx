import React, { useState } from 'react';
import { ExternalLink, Layers, ArrowUpRight } from 'lucide-react';
import type { Project } from '../types';

interface ProjectsSectionProps {
  projects: Project[];
}

export const ProjectsSection: React.FC<ProjectsSectionProps> = ({ projects }) => {
  const [selectedCategory, setSelectedCategory] = useState('All');

  const categories = ['All', 'Full-Stack', 'AI & Agents', 'Systems', 'Frontend'];

  const filteredProjects = selectedCategory === 'All' 
    ? projects 
    : projects.filter(p => p.category.toLowerCase() === selectedCategory.toLowerCase());

  return (
    <section id="projects" className="py-24 relative overflow-hidden">
      {/* Glow highlight */}
      <div className="absolute top-1/2 right-0 w-[500px] h-[500px] bg-blue-600/10 blur-[130px] rounded-full pointer-events-none" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section Header */}
        <div className="flex flex-col md:flex-row md:items-end justify-between mb-12 gap-6">
          <div>
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-blue-950/80 border border-blue-800/40 text-blue-400 text-xs font-mono uppercase tracking-wider mb-4">
              <Layers className="w-3.5 h-3.5" />
              <span>Production Work & Systems</span>
            </div>
            <h2 className="text-3xl sm:text-5xl font-extrabold text-white tracking-tight">
              Featured Case Studies & Projects
            </h2>
            <p className="text-slate-400 text-sm sm:text-base mt-2 max-w-xl">
              High-impact solutions engineered for performance, reliability, and real-world scale.
            </p>
          </div>

          {/* Filter Pills */}
          <div className="flex flex-wrap gap-1.5 p-1.5 rounded-2xl bg-slate-900/80 border border-blue-900/30 backdrop-blur-md">
            {categories.map(cat => (
              <button
                key={cat}
                onClick={() => setSelectedCategory(cat)}
                className={`px-3.5 py-1.5 rounded-xl text-xs sm:text-sm font-medium transition-all duration-200 cursor-pointer ${
                  selectedCategory === cat
                    ? 'bg-blue-600 text-white shadow-md shadow-blue-600/30 font-semibold'
                    : 'text-slate-400 hover:text-white hover:bg-slate-800/60'
                }`}
              >
                {cat}
              </button>
            ))}
          </div>
        </div>

        {/* Project Cards Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredProjects.map((project) => (
            <div
              key={project.id}
              className="flex flex-col rounded-2xl overflow-hidden bg-[#0a1226]/80 border border-blue-900/30 hover:border-blue-500/50 hover:shadow-2xl hover:shadow-blue-900/20 hover:-translate-y-1.5 transition-all duration-300 group"
            >
              {/* Card Image Container */}
              <div className="relative h-52 overflow-hidden bg-slate-950">
                <img
                  src={project.image}
                  alt={project.title}
                  className="w-full h-full object-cover object-center group-hover:scale-105 transition-transform duration-500 opacity-80 group-hover:opacity-100"
                  loading="lazy"
                />
                <div className="absolute inset-0 bg-gradient-to-t from-[#0a1226] via-[#0a1226]/40 to-transparent" />
                
                {/* Category Badge */}
                <div className="absolute top-3.5 left-3.5">
                  <span className="px-2.5 py-1 rounded-md text-[11px] font-semibold bg-blue-950/90 text-blue-300 border border-blue-800/50 backdrop-blur-md">
                    {project.category}
                  </span>
                </div>

                {/* Metrics Highlight Pill */}
                {project.metrics && (
                  <div className="absolute bottom-3 left-3 right-3">
                    <div className="px-2.5 py-1 rounded-md bg-slate-950/85 border border-blue-800/40 text-[11px] font-mono text-cyan-300 backdrop-blur-md truncate">
                      ⚡ {project.metrics}
                    </div>
                  </div>
                )}
              </div>

              {/* Card Body */}
              <div className="p-6 flex-1 flex flex-col justify-between">
                <div>
                  <h3 className="text-xl font-bold text-white group-hover:text-blue-300 transition-colors mb-2 tracking-tight flex items-center justify-between">
                    <span>{project.title}</span>
                    <ArrowUpRight className="w-4 h-4 text-slate-500 group-hover:text-blue-400 group-hover:translate-x-0.5 group-hover:-translate-y-0.5 transition-all" />
                  </h3>

                  <p className="text-xs text-blue-400 font-medium mb-3">
                    {project.tagline}
                  </p>

                  <p className="text-sm text-slate-300/80 leading-relaxed mb-5 line-clamp-3">
                    {project.description}
                  </p>
                </div>

                <div>
                  {/* Tech Tags */}
                  <div className="flex flex-wrap gap-1.5 mb-5">
                    {project.tags.map(tag => (
                      <span
                        key={tag}
                        className="px-2 py-0.5 text-[11px] font-mono rounded bg-slate-900/90 text-slate-300 border border-slate-800"
                      >
                        {tag}
                      </span>
                    ))}
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-3 pt-4 border-t border-blue-950/80">
                    <a
                      href={project.github_url}
                      target="_blank"
                      rel="noreferrer"
                      className="flex-1 flex items-center justify-center gap-1.5 py-2 px-3 rounded-lg bg-slate-900 hover:bg-slate-800 text-xs font-semibold text-slate-300 hover:text-white border border-blue-900/30 transition-colors"
                    >
                      <svg className="w-3.5 h-3.5 fill-current" viewBox="0 0 24 24">
                        <path fillRule="evenodd" clipRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" />
                      </svg>
                      <span>Repository</span>
                    </a>

                    <a
                      href={project.demo_url}
                      target="_blank"
                      rel="noreferrer"
                      className="flex-1 flex items-center justify-center gap-1.5 py-2 px-3 rounded-lg bg-blue-600/20 hover:bg-blue-600/30 text-xs font-semibold text-blue-300 hover:text-white border border-blue-500/30 transition-colors"
                    >
                      <ExternalLink className="w-3.5 h-3.5" />
                      <span>Live Demo</span>
                    </a>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};
