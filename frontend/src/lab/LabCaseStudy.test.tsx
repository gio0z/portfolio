import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { LabCaseStudy } from './LabCaseStudy';
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

function renderCaseStudy(slug: string, initialProject?: LabProject) {
  return render(
    <MemoryRouter initialEntries={[`/lab/${slug}`]}>
      <Routes>
        <Route path="/lab/:slug" element={<LabCaseStudy initialProject={initialProject} />} />
      </Routes>
    </MemoryRouter>,
  );
}

function mockProjectResponse(project: LabProject, status = 200) {
  return vi.spyOn(globalThis, 'fetch').mockResolvedValue(
    new Response(JSON.stringify(project), {
      status,
      headers: { 'Content-Type': 'application/json' },
    }),
  );
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe('LabCaseStudy', () => {
  it('renders rationale, before/after evidence, media, and a sandboxed preview', () => {
    const { container } = renderCaseStudy('checkout-redesign', makeProject());

    expect(screen.getByRole('heading', { name: 'Checkout Redesign' })).toBeInTheDocument();
    expect(
      screen.getByTestId('lab-disclaimer'),
    ).toHaveTextContent(LAB_DISCLAIMER);
    expect(container.querySelector('main')?.firstElementChild).toHaveTextContent(
      LAB_DISCLAIMER,
    );
    expect(screen.getByText('A faster, calmer checkout.')).toBeInTheDocument();
    expect(screen.getByText('Before')).toBeInTheDocument();
    expect(screen.getByText('After')).toBeInTheDocument();
    expect(screen.getByRole('img', { name: /checkout redesign cover/i })).toBeInTheDocument();

    const preview = screen.getByTestId('lab-preview');
    expect(preview.tagName).toBe('IFRAME');
    expect(preview).toHaveAttribute('src', 'https://lab.example/checkout');
    expect(preview.getAttribute('sandbox')).toBe('allow-scripts allow-forms');
    expect(preview.getAttribute('sandbox')).not.toContain('allow-same-origin');
  });

  it('fetches the case study by slug from the public API', async () => {
    const fetchSpy = mockProjectResponse(makeProject());

    renderCaseStudy('checkout-redesign');

    expect(
      await screen.findByRole('heading', { name: 'Checkout Redesign' }),
    ).toBeInTheDocument();
    expect(fetchSpy).toHaveBeenCalledWith('/api/lab/projects/checkout-redesign');
  });

  it('renders not-found for a missing slug', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ error: 'project not found' }), {
        status: 404,
        headers: { 'Content-Type': 'application/json' },
      }),
    );

    renderCaseStudy('missing-slug');

    expect(await screen.findByRole('heading', { name: /project not found/i })).toBeInTheDocument();
    expect(screen.queryByTestId('lab-preview')).not.toBeInTheDocument();
  });

  it('never renders unpublished records', async () => {
    mockProjectResponse(makeProject({ status: 'DRAFT', title: 'Hidden Draft' }));

    renderCaseStudy('hidden-draft');

    expect(await screen.findByRole('heading', { name: /project not found/i })).toBeInTheDocument();
    expect(screen.queryByText('Hidden Draft')).not.toBeInTheDocument();
  });

  it('treats an unpublished initial project as not-found', () => {
    renderCaseStudy(
      'hidden-draft',
      makeProject({ status: 'IN_REVIEW', title: 'Hidden Review' }),
    );

    expect(screen.getByRole('heading', { name: /project not found/i })).toBeInTheDocument();
    expect(screen.queryByText('Hidden Review')).not.toBeInTheDocument();
  });

  it('hides the live preview when no demo url is published', () => {
    renderCaseStudy('checkout-redesign', makeProject({ live_demo_url: undefined }));

    expect(screen.queryByTestId('lab-preview')).not.toBeInTheDocument();
    expect(screen.getByText(/live preview is not available/i)).toBeInTheDocument();
  });

  it('renders source site links as outbound references with an origin-vs-redesign verdict', () => {
    renderCaseStudy(
      'checkout-redesign',
      makeProject({
        source_urls: ['https://acme.example/shop', 'https://acme.example/checkout'],
      }),
    );

    expect(screen.getByRole('heading', { name: 'Source site' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Checkout Redesign' })).toBeInTheDocument();

    const first = screen.getByRole('link', { name: 'https://acme.example/shop' });
    expect(first).toHaveAttribute('href', 'https://acme.example/shop');
    expect(first).toHaveAttribute('target', '_blank');
    expect(first).toHaveAttribute('rel', 'noopener noreferrer');

    const second = screen.getByRole('link', { name: 'https://acme.example/checkout' });
    expect(second).toHaveAttribute('href', 'https://acme.example/checkout');
    expect(second).toHaveAttribute('target', '_blank');
    expect(second).toHaveAttribute('rel', 'noopener noreferrer');

    const verdict = screen.getByTestId('lab-source-site-verdict');
    expect(verdict).toHaveTextContent(/original third-party product/i);
    expect(verdict).toHaveTextContent(/independent redesign concept/i);
    expect(verdict).toHaveTextContent(/not affiliated with or endorsed by/i);
  });

  it('omits the source site section when source urls are empty', () => {
    renderCaseStudy('checkout-redesign', makeProject({ source_urls: [] }));

    expect(screen.queryByTestId('lab-source-site')).not.toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Source site' })).not.toBeInTheDocument();
    expect(screen.queryByTestId('lab-source-site-verdict')).not.toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Checkout Redesign' })).toBeInTheDocument();
  });

  it('renders the case study when source urls are absent', () => {
    renderCaseStudy('checkout-redesign', makeProject());

    expect(screen.getByRole('heading', { name: 'Checkout Redesign' })).toBeInTheDocument();
    expect(screen.queryByTestId('lab-source-site')).not.toBeInTheDocument();
    expect(screen.queryByText(/source site/i)).not.toBeInTheDocument();
  });

  it('never renders a non-http source url as a clickable link', () => {
    const { container } = renderCaseStudy(
      'checkout-redesign',
      makeProject({ source_urls: ['javascript:alert(1)', 'https://acme.example/shop'] }),
    );

    expect(screen.getByRole('link', { name: 'https://acme.example/shop' })).toBeInTheDocument();
    expect(screen.queryByText('javascript:alert(1)')).not.toBeInTheDocument();
    expect(
      Array.from(container.querySelectorAll('a')).some((anchor) =>
        (anchor.getAttribute('href') ?? '').startsWith('javascript:'),
      ),
    ).toBe(false);
    expect(screen.getAllByRole('link')).toHaveLength(2);
  });
});
