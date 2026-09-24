import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import {
  LAB_DISCLAIMER,
  fetchLabProject,
  isPublishedProject,
  type LabProject,
} from './LabApi';

interface LabCaseStudyProps {
  initialProject?: LabProject;
}

type CaseStudyState =
  | { kind: 'loading' }
  | { kind: 'ready'; project: LabProject }
  | { kind: 'missing'; message: string };

function stateForInitial(initialProject: LabProject | undefined): CaseStudyState {
  if (initialProject === undefined) {
    return { kind: 'loading' };
  }
  if (!isPublishedProject(initialProject)) {
    return { kind: 'missing', message: 'Project not found.' };
  }
  return { kind: 'ready', project: initialProject };
}

export function LabCaseStudy({ initialProject }: LabCaseStudyProps) {
  const { slug = '' } = useParams();
  const [state, setState] = useState<CaseStudyState>(() => stateForInitial(initialProject));

  useEffect(() => {
    if (initialProject !== undefined) {
      return;
    }
    let cancelled = false;
    fetchLabProject(slug)
      .then((project) => {
        if (!cancelled) {
          setState({ kind: 'ready', project });
        }
      })
      .catch((error: Error) => {
        if (!cancelled) {
          setState({ kind: 'missing', message: error.message });
        }
      });
    return () => {
      cancelled = true;
    };
  }, [slug, initialProject]);

  if (state.kind === 'loading') {
    return (
      <main className="min-h-screen bg-[#F8F9FA] text-[#18181B]">
        <p data-testid="lab-disclaimer" className="bg-amber-50 px-6 py-3 text-sm text-amber-900">
          {LAB_DISCLAIMER}
        </p>
        <div className="mx-auto max-w-4xl px-6 py-10">
          <h1 className="text-3xl font-bold tracking-tight">Design Lab</h1>
          <p className="mt-2 text-zinc-600">Loading case study…</p>
        </div>
      </main>
    );
  }
  if (state.kind === 'missing') {
    return (
      <main className="min-h-screen bg-[#F8F9FA] text-[#18181B]">
        <p data-testid="lab-disclaimer" className="bg-amber-50 px-6 py-3 text-sm text-amber-900">
          {LAB_DISCLAIMER}
        </p>
        <div className="mx-auto max-w-4xl px-6 py-10">
          <h1 className="text-3xl font-bold tracking-tight">Project not found</h1>
          <p className="mt-2 text-zinc-600">{state.message}</p>
          <Link to="/lab" className="mt-4 inline-block font-semibold text-blue-700 hover:underline">
            Back to the catalog
          </Link>
        </div>
      </main>
    );
  }

  const project = state.project;
  const rationale =
    project.summary !== undefined && project.summary !== ''
      ? project.summary
      : 'Redesign rationale has not been published yet.';
  const sourceUrls = (project.source_urls ?? []).filter((url) => {
    try {
      const parsed = new URL(url);
      return parsed.protocol === 'http:' || parsed.protocol === 'https:';
    } catch {
      return false;
    }
  });

  return (
    <main className="min-h-screen bg-[#F8F9FA] text-[#18181B]">
      <p data-testid="lab-disclaimer" className="bg-amber-50 px-6 py-3 text-sm text-amber-900">
        {project.disclaimer === '' ? LAB_DISCLAIMER : project.disclaimer}
      </p>
      <article className="mx-auto max-w-4xl px-6 py-10">
        <Link to="/lab" className="text-sm font-semibold text-blue-700 hover:underline">
          ← Back to the catalog
        </Link>
        <h1 className="mt-4 text-3xl font-bold tracking-tight">{project.title}</h1>
        <p className="mt-2 text-zinc-600">Original product: {project.original_product}</p>
        {(project.focus !== undefined && project.focus.length > 0) ||
        (project.platforms !== undefined && project.platforms.length > 0) ? (
          <p className="mt-1 text-sm text-zinc-600">
            {[project.focus?.join(', '), project.platforms?.join(', ')]
              .filter((part) => part !== undefined && part !== '')
              .join(' · ')}
          </p>
        ) : null}

        <section aria-label="Rationale" className="mt-8">
          <h2 className="text-xl font-semibold">Why this redesign</h2>
          <p className="mt-2 text-zinc-700">{rationale}</p>
        </section>

        <section aria-label="Before and after" className="mt-8 grid gap-4 sm:grid-cols-2">
          <figure className="rounded-xl border border-zinc-200 bg-white p-4">
            <figcaption className="text-sm font-semibold">Before</figcaption>
            <p className="mt-1 text-sm text-zinc-600">
              The original {project.original_product} experience this concept starts from.
            </p>
          </figure>
          <figure className="rounded-xl border border-zinc-200 bg-white p-4">
            <figcaption className="text-sm font-semibold">After</figcaption>
            <p className="mt-1 text-sm text-zinc-600">
              The redesigned flow: {project.focus?.join(', ') ?? 'see the live preview below'}.
            </p>
          </figure>
        </section>

        {sourceUrls.length > 0 && (
          <section aria-label="Source site" data-testid="lab-source-site" className="mt-8">
            <h2 className="text-xl font-semibold">Source site</h2>
            <p data-testid="lab-source-site-verdict" className="mt-2 text-sm text-zinc-600">
              These links point to the original third-party product this concept starts from. This
              case study is an independent redesign concept: it is not the original product and is
              not affiliated with or endorsed by the owner of the linked site.
            </p>
            <ul className="mt-3 space-y-1">
              {sourceUrls.map((url) => (
                <li key={url}>
                  <a
                    href={url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-sm font-semibold text-blue-700 hover:underline"
                  >
                    {url}
                  </a>
                </li>
              ))}
            </ul>
          </section>
        )}

        {project.cover_image !== undefined && project.cover_image !== '' && (
          <section aria-label="Media" className="mt-8">
            <h2 className="text-xl font-semibold">Media</h2>
            <img
              src={project.cover_image}
              alt={`${project.title} cover`}
              className="mt-3 w-full rounded-xl border border-zinc-200 object-cover"
              loading="lazy"
            />
          </section>
        )}

        <section aria-label="Live preview" className="mt-8">
          <h2 className="text-xl font-semibold">Live preview</h2>
          {project.live_demo_url !== undefined && project.live_demo_url !== '' ? (
            <>
              <p className="mt-1 text-sm text-zinc-600">
                Sandboxed preview of the published artifact. Scripts run without same-origin
                access.
              </p>
              <div className="mt-3 overflow-hidden rounded-xl border border-zinc-200 bg-white">
                <iframe
                  data-testid="lab-preview"
                  title={`${project.title} live preview`}
                  src={project.live_demo_url}
                  sandbox="allow-scripts allow-forms"
                  className="h-[480px] w-full"
                />
              </div>
            </>
          ) : (
            <p className="mt-2 text-sm text-zinc-600">
              A live preview is not available for this case study yet.
            </p>
          )}
        </section>
      </article>
    </main>
  );
}

export default LabCaseStudy;
