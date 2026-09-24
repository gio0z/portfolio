import React, { useCallback, useRef, useState } from 'react';
import { DesignLabCard } from './DesignLabCard';
import type { DesignLabProject } from '../types';

export const DESIGN_LAB_ENDPOINT = '/api/portfolio/design-lab';
export const DEFAULT_LAB_ORIGIN = '/lab';

type WorkTabId = 'engineering' | 'lab';

interface WorkTabsProps {
  /** Engineering Work content (the layered coverflow). Rendered under the default tab. */
  children: React.ReactNode;
  /** Injected published Lab records. When provided, no network request is made. */
  labProjects?: DesignLabProject[];
  /** Configured Lab origin for catalog and case-study fallback links. */
  labOrigin?: string;
}

function normalizeLabOrigin(origin: string | undefined): string {
  if (!origin || origin.length === 0) return DEFAULT_LAB_ORIGIN;
  return origin.replace(/\/+$/, '') || DEFAULT_LAB_ORIGIN;
}

/**
 * Two semantically distinct views under one section: Engineering Work
 * (repository-backed coverflow, default-selected) and Design Lab
 * (published Lab registry records with the mandatory legal disclaimer).
 */
export const WorkTabs: React.FC<WorkTabsProps> = ({
  children,
  labProjects: injectedLabProjects,
  labOrigin,
}) => {
  const [activeTab, setActiveTab] = useState<WorkTabId>('engineering');
  const [fetchedLabProjects, setFetchedLabProjects] = useState<DesignLabProject[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchStarted = useRef(false);
  const origin = normalizeLabOrigin(labOrigin);
  const labProjects = injectedLabProjects ?? fetchedLabProjects;

  const loadLabProjects = useCallback(async () => {
    if (injectedLabProjects !== undefined) return;
    if (fetchStarted.current) return;
    fetchStarted.current = true;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(DESIGN_LAB_ENDPOINT);
      if (!res.ok) {
        throw new Error(`Design Lab request failed with status ${res.status}`);
      }
      const data = (await res.json()) as unknown;
      setFetchedLabProjects(Array.isArray(data) ? (data as DesignLabProject[]) : []);
    } catch {
      setError('Design Lab is unavailable right now. Please try again later.');
      setFetchedLabProjects(null);
    } finally {
      setLoading(false);
    }
  }, [injectedLabProjects]);

  const selectTab = (tab: WorkTabId) => {
    setActiveTab(tab);
    if (tab === 'lab') {
      void loadLabProjects();
    }
  };

  const tabClass = (selected: boolean) =>
    `px-4 py-2 rounded-full text-xs font-semibold font-mono transition-all cursor-pointer ${
      selected
        ? 'bg-zinc-900 text-white shadow-md'
        : 'text-zinc-600 hover:bg-zinc-200/70 hover:text-zinc-900'
    }`;

  return (
    <div>
      <div
        role="tablist"
        aria-label="Work views"
        className="flex flex-wrap gap-2 p-1.5 rounded-full bg-zinc-200/80 border border-zinc-300/60 w-fit mb-8"
      >
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'engineering'}
          aria-controls="work-panel-engineering"
          id="work-tab-engineering"
          onClick={() => selectTab('engineering')}
          className={tabClass(activeTab === 'engineering')}
        >
          Engineering Work
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'lab'}
          aria-controls="work-panel-lab"
          id="work-tab-lab"
          onClick={() => selectTab('lab')}
          className={tabClass(activeTab === 'lab')}
        >
          Design Lab
        </button>
      </div>

      {activeTab === 'engineering' ? (
        <div
          role="tabpanel"
          id="work-panel-engineering"
          aria-labelledby="work-tab-engineering"
        >
          {children}
        </div>
      ) : (
        <div role="tabpanel" id="work-panel-lab" aria-labelledby="work-tab-lab">
          {loading ? (
            <p className="text-sm text-zinc-500 font-mono">Loading Design Lab…</p>
          ) : error ? (
            <p role="alert" className="text-sm text-red-700 bg-red-50 border border-red-200 rounded-xl p-4">
              {error}
            </p>
          ) : labProjects && labProjects.length > 0 ? (
            <>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 sm:gap-8">
                {labProjects.map((project, index) => (
                  <DesignLabCard
                    key={project.id}
                    project={project}
                    labOrigin={origin}
                    index={index}
                  />
                ))}
              </div>
              <div className="mt-8 text-center">
                <a
                  href={origin}
                  className="inline-flex items-center justify-center gap-2 py-3 px-6 rounded-xl bg-zinc-900 hover:bg-zinc-800 text-white font-semibold text-xs transition-colors"
                >
                  <span>Explore Design Lab</span>
                </a>
              </div>
            </>
          ) : (
            <p className="text-sm text-zinc-500 font-mono">
              No published Design Lab work yet.
            </p>
          )}
        </div>
      )}
    </div>
  );
};

export default WorkTabs;
