import React, { useState } from 'react';
import { ArrowDownRight, ExternalLink } from 'lucide-react';
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
    <section id="projects" className="py-24 sm:py-32 px-4 sm:px-8 max-w-7xl mx-auto">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-end justify-between mb-14 sm:mb-18 gap-6">
        <div>
          <div className="inline-flex items-center gap-2 text-xs font-semibold tracking-wider uppercase text-zinc-500 mb-4 font-mono">
            <span className="flex items-center justify-center w-4 h-4 rounded bg-blue-600 text-white">
              <ArrowDownRight className="w-3 h-3" />
            </span>
            <span>Case Studies</span>
          </div>
          <h2 className="text-4xl sm:text-5xl lg:text-6xl font-extrabold tracking-tight text-zinc-900 leading-[1.1]">
            <span>Featured</span> <br />
            <span className="text-zinc-400 font-bold">Engineering Work</span>
          </h2>
        </div>

        {/* Filter Pills */}
        <div className="flex flex-wrap gap-2 p-1.5 rounded-full bg-zinc-200/80 border border-zinc-300/60">
          {categories.map((cat) => (
            <button
              key={cat}
              onClick={() => setSelectedCategory(cat)}
              className={`px-4 py-2 rounded-full text-xs sm:text-sm font-semibold transition-all duration-200 cursor-pointer ${
                selectedCategory === cat
                  ? 'bg-blue-600 text-white shadow-sm'
                  : 'text-zinc-600 hover:text-zinc-900 hover:bg-zinc-100'
              }`}
            >
              {cat}
            </button>
          ))}
        </div>
      </div>

      {/* Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 sm:gap-8">
        {filteredProjects.map((project) => (
          <div
            key={project.id}
            className="bg-white rounded-[28px] overflow-hidden border border-zinc-200/90 shadow-sm hover:shadow-2xl hover:border-blue-500/40 hover:-translate-y-1.5 transition-all duration-300 flex flex-col justify-between group"
          >
            {/* Image Preview Container */}
            <div className="relative h-56 overflow-hidden bg-zinc-100">
              <img
                src={project.image}
                alt={project.title}
                className="w-full h-full object-cover object-center group-hover:scale-105 transition-transform duration-500"
                loading="lazy"
              />
              <div className="absolute top-4 left-4">
                <span className="px-3 py-1 rounded-full text-xs font-semibold bg-white/95 text-zinc-900 shadow-sm backdrop-blur-md">
                  {project.category}
                </span>
              </div>
            </div>

            {/* Content Body */}
            <div className="p-6 sm:p-8 flex-1 flex flex-col justify-between">
              <div>
                <h3 className="text-xl font-bold text-zinc-900 group-hover:text-blue-600 transition-colors mb-2 tracking-tight flex items-center justify-between">
                  <span>{project.title}</span>
                  <span className="w-7 h-7 rounded-full bg-zinc-100 group-hover:bg-blue-600 group-hover:text-white flex items-center justify-center text-zinc-600 transition-colors shrink-0">
                    <ArrowDownRight className="w-4 h-4" />
                  </span>
                </h3>

                <p className="text-xs font-semibold text-blue-600 mb-3">
                  {project.tagline}
                </p>

                <p className="text-sm text-zinc-600 leading-relaxed mb-6 line-clamp-3">
                  {project.description}
                </p>
              </div>

              <div>
                {/* Metrics Pill */}
                {project.metrics && (
                  <div className="mb-4 px-3 py-1.5 rounded-xl bg-blue-50/80 border border-blue-100 text-xs font-mono font-medium text-blue-800">
                    ⚡ {project.metrics}
                  </div>
                )}

                {/* Tech Tags */}
                <div className="flex flex-wrap gap-1.5 mb-6">
                  {project.tags.map((tag) => (
                    <span
                      key={tag}
                      className="px-2.5 py-0.5 text-xs font-mono rounded-md bg-zinc-100 text-zinc-700 font-medium"
                    >
                      {tag}
                    </span>
                  ))}
                </div>

                {/* Links */}
                <div className="flex items-center gap-3 pt-4 border-t border-zinc-100">
                  <a
                    href={project.github_url}
                    target="_blank"
                    rel="noreferrer"
                    className="flex-1 flex items-center justify-center gap-1.5 py-2.5 px-3 rounded-full bg-zinc-100 hover:bg-zinc-200 text-xs font-semibold text-zinc-800 transition-colors"
                  >
                    <span>Repository</span>
                  </a>

                  <a
                    href={project.demo_url}
                    target="_blank"
                    rel="noreferrer"
                    className="flex-1 flex items-center justify-center gap-1.5 py-2.5 px-3 rounded-full bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold shadow-sm transition-colors"
                  >
                    <span>Live Demo</span>
                    <ExternalLink className="w-3.5 h-3.5" />
                  </a>
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
};
