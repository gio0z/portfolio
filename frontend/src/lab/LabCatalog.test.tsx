import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { LabCatalog } from './LabCatalog';
import { LAB_DISCLAIMER, type LabProject } from './LabApi';

function makeProject(overrides: Partial<LabProject> = {}): LabProject {
  return {
    id: 'lab-1',
    slug: 'checkout-redesign',
    title: 'Checkout Redesign',
    original_product: 'Acme Shop',
    disclaimer: LAB_DISCLAIMER,
    focus: ['navigation', 'checkout flow'],
    platforms: ['web'],
    status: 'PUBLISHED',
    featured: false,
    created_at: '2026-09-01T08:00:00Z',
    updated_at: '2026-09-03T10:30:00Z',
    cover_image: 'https://cdn.example/cover.png',
    summary: 'A faster, calmer checkout.',
    live_demo_url: 'https://lab.example/checkout',
    ...overrides,
  };
}

function renderCatalog(initialProjects?: LabProject[]) {
  return render(
    <MemoryRouter>
      <LabCatalog initialProjects={initialProjects} />
    </MemoryRouter>,
  );
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe('LabCatalog', () => {
  it('renders only published records', () => {
    renderCatalog([
      makeProject(),
      makeProject({ id: 'lab-2', slug: 'draft-proj', title: 'Draft Project', status: 'DRAFT' }),
      makeProject({
        id: 'lab-3',
        slug: 'in-review-proj',
        title: 'In Review Project',
        status: 'IN_REVIEW',
      }),
    ]);

    expect(screen.getByText('Checkout Redesign')).toBeInTheDocument();
    expect(screen.queryByText('Draft Project')).not.toBeInTheDocument();
    expect(screen.queryByText('In Review Project')).not.toBeInTheDocument();
  });

  it('shows the mandatory disclaimer above the fold', () => {
    const { container } = renderCatalog([makeProject()]);

    const disclaimer = screen.getByTestId('lab-disclaimer');
    expect(disclaimer).toHaveTextContent(LAB_DISCLAIMER);
    expect(container.querySelector('main')?.firstElementChild).toBe(disclaimer);
  });

  it('shows card metadata and links to the case study', () => {
    renderCatalog([makeProject()]);

    expect(screen.getByText(/Acme Shop/)).toBeInTheDocument();
    expect(screen.getByText(/navigation/)).toBeInTheDocument();
    expect(screen.getByText(/web/)).toBeInTheDocument();
    expect(screen.getByText('PUBLISHED')).toBeInTheDocument();
    expect(
      screen.getByRole('img', { name: /checkout redesign cover/i }),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /view case study/i })).toHaveAttribute(
      'href',
      '/lab/checkout-redesign',
    );
  });

  it('fetches the catalog from the public API and drops unpublished records', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(
        JSON.stringify([
          makeProject(),
          makeProject({ id: 'lab-9', slug: 'leaked-draft', title: 'Leaked Draft', status: 'DRAFT' }),
        ]),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    );

    render(
      <MemoryRouter>
        <LabCatalog />
      </MemoryRouter>,
    );

    expect(await screen.findByText('Checkout Redesign')).toBeInTheDocument();
    expect(screen.queryByText('Leaked Draft')).not.toBeInTheDocument();
    expect(fetchSpy).toHaveBeenCalledWith('/api/lab/projects');
  });

  it('shows an empty state when nothing is published', () => {
    renderCatalog([makeProject({ status: 'DRAFT', title: 'Hidden Draft' })]);

    expect(screen.getByText(/no published redesigns/i)).toBeInTheDocument();
  });

  it('shows an error state when the catalog cannot load', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('offline'));

    render(
      <MemoryRouter>
        <LabCatalog />
      </MemoryRouter>,
    );

    expect(await screen.findByRole('alert')).toHaveTextContent(/could not load/i);
  });
});
