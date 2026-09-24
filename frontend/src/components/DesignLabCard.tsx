import React from 'react';
import { ExternalLink } from 'lucide-react';
import type { DesignLabProject } from '../types';

interface DesignLabCardProps {
  project: DesignLabProject;
  labOrigin?: string;
  index: number;
}

/**
 * Published Design Lab card. Reuses the portfolio card layout without the
 * engineering metadata contract: no GitHub repository link is ever rendered.
 */
export const DesignLabCard: React.FC<DesignLabCardProps> = ({ project, labOrigin, index }) => {
  const origin = (labOrigin ?? '').replace(/\/+$/, '');
  const caseStudyHref =
    project.case_study_url && project.case_study_url.length > 0
      ? project.case_study_url
      : `${origin || '/lab'}/${project.slug}`;

  return (
    <article
      className="bg-white rounded-[28px] overflow-hidden border border-zinc-200/90 shadow-sm hover:shadow-2xl hover:border-blue-500/40 hover:-translate-y-1.5 transition-all duration-300 flex flex-col"
      data-testid={`design-lab-card-${project.slug}`}
    >
      {project.cover_image ? (
        <img
          src={project.cover_image}
          alt={`${project.title} cover`}
          className="w-full h-48 object-cover"
          loading="lazy"
        />
      ) : (
        <div
          className="w-full h-48 bg-gradient-to-br from-blue-600 via-indigo-600 to-zinc-900 flex items-center justify-center"
          aria-hidden="true"
        >
          <span className="text-6xl font-bold text-white/90 tracking-tighter select-none">
            {index + 1}
          </span>
        </div>
      )}

      <div className="p-6 flex flex-col flex-1">
        <div className="flex flex-wrap items-center gap-2 mb-3">
          <span className="px-3 py-1 rounded-full bg-blue-50 text-blue-700 text-xs font-semibold font-mono border border-blue-200/60">
            {project.original_product}
          </span>
          {project.platforms?.map((platform) => (
            <span
              key={platform}
              className="px-3 py-1 rounded-md bg-zinc-100 text-zinc-700 text-xs font-mono font-medium"
            >
              {platform}
            </span>
          ))}
        </div>

        <h3 className="text-xl font-extrabold text-zinc-900 mb-2 tracking-tight">
          {project.title}
        </h3>

        {project.summary ? (
          <p className="text-sm text-zinc-600 leading-relaxed mb-4">{project.summary}</p>
        ) : null}

        {project.focus && project.focus.length > 0 ? (
          <div className="flex flex-wrap gap-2 mb-4">
            {project.focus.map((item) => (
              <span
                key={item}
                className="px-3 py-1 rounded-md bg-zinc-100 text-zinc-700 text-xs font-mono font-medium"
              >
                {item}
              </span>
            ))}
          </div>
        ) : null}

        <p className="text-xs text-zinc-500 leading-relaxed mb-6 mt-auto">{project.disclaimer}</p>

        <a
          href={caseStudyHref}
          target={project.case_study_url ? '_blank' : undefined}
          rel={project.case_study_url ? 'noreferrer' : undefined}
          className="w-full flex items-center justify-center gap-2 py-3 px-4 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs shadow-md shadow-blue-600/20 transition-colors"
        >
          <span>View Case Study</span>
          <ExternalLink className="w-3.5 h-3.5" />
        </a>
      </div>
    </article>
  );
};

export default DesignLabCard;
