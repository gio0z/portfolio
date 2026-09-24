import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  LAB_DISCLAIMER,
  fetchLabCatalog,
  isPublishedProject,
  type LabProject,
} from './LabApi';

interface LabCatalogProps {
  initialProjects?: LabProject[];
}

export function LabCatalog({ initialProjects }: LabCatalogProps) {
  const [projects, setProjects] = useState<LabProject[]>(() =>
    initialProjects === undefined ? [] : initialProjects.filter(isPublishedProject),
  );
  const [loading, setLoading] = useState(initialProjects === undefined);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (initialProjects !== undefined) {
      return;
    }
    let cancelled = false;
    fetchLabCatalog()
      .then((items) => {
        if (!cancelled) {
          setProjects(items);
          setLoading(false);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setError('Could not load the redesign catalog. Please try again later.');
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [initialProjects]);

  return (
    <main className="min-h-screen bg-[#F8F9FA] text-[#18181B]">
      <p data-testid="lab-disclaimer" className="bg-amber-50 px-6 py-3 text-sm text-amber-900">
        {LAB_DISCLAIMER}
      </p>
      <div className="mx-auto max-w-6xl px-6 py-10">
        <h1 className="text-3xl font-bold tracking-tight">Design Lab</h1>
        <p className="mt-2 text-zinc-600">
          Independent redesign concepts and interface experiments.
        </p>
        {loading ? (
          <p className="mt-8 text-zinc-600">Loading redesigns…</p>
        ) : error !== null ? (
          <p role="alert" className="mt-8 text-red-700">
            {error}
          </p>
        ) : projects.length === 0 ? (
          <p className="mt-8 text-zinc-600">No published redesigns yet. Check back soon.</p>
        ) : (
          <ul className="mt-8 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {projects.map((project) => (
              <li
                key={project.id}
                className="overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-sm"
              >
                {project.cover_image !== undefined && project.cover_image !== '' && (
                  <img
                    src={project.cover_image}
                    alt={`${project.title} cover`}
                    className="h-44 w-full object-cover"
                    loading="lazy"
                  />
                )}
                <div className="flex flex-col gap-2 p-5">
                  <h2 className="text-lg font-semibold">{project.title}</h2>
                  <p className="text-sm text-zinc-600">
                    Original product: {project.original_product}
                  </p>
                  {project.focus !== undefined && project.focus.length > 0 && (
                    <p className="text-sm text-zinc-600">
                      Redesign focus: {project.focus.join(', ')}
                    </p>
                  )}
                  {project.platforms !== undefined && project.platforms.length > 0 && (
                    <p className="text-sm text-zinc-600">
                      Platform: {project.platforms.join(', ')}
                    </p>
                  )}
                  <p className="text-xs font-medium uppercase tracking-wide text-zinc-500">
                    {project.status}
                  </p>
                  <Link
                    to={`/lab/${project.slug}`}
                    className="mt-1 text-sm font-semibold text-blue-700 hover:underline"
                  >
                    View case study
                  </Link>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </main>
  );
}

export default LabCatalog;
