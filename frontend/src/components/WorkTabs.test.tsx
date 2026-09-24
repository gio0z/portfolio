import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { WorkTabs } from './WorkTabs';
import { CoverflowSection } from './CoverflowSection';
import type { DesignLabProject, Project } from '../types';

const REQUIRED_DISCLAIMER =
  'Independent redesign concept. Not affiliated with or endorsed by the original company.';

const engineeringProjects: Project[] = [
  {
    id: 'proj-1',
    title: 'Realtime Sync Engine',
    tagline: 'Conflict-free collaboration',
    description: 'A production sync engine backed by a GitHub repository.',
    category: 'Systems',
    tags: ['go', 'websockets'],
    featured: true,
    github_url: 'https://github.com/gio0z/sync-engine',
    demo_url: 'https://demo.example.com/sync',
    image: '',
    metrics: '99.99% uptime',
  },
];

const labRecords: DesignLabProject[] = [
  {
    id: 'lab_1',
    slug: 'checkout-redesign',
    title: 'Checkout Redesign',
    original_product: 'Acme Checkout',
    disclaimer: REQUIRED_DISCLAIMER,
    focus: ['navigation', 'information architecture'],
    platforms: ['web'],
    status: 'PUBLISHED',
    featured: true,
    summary: 'A streamlined checkout flow concept.',
    case_study_url: 'https://lab.example.com/checkout-redesign',
  },
  {
    id: 'lab_2',
    slug: 'analytics-widget',
    title: 'Analytics Widget',
    original_product: 'Metrics Hub',
    disclaimer: REQUIRED_DISCLAIMER,
    focus: ['data visualization'],
    platforms: ['mobile'],
    status: 'PUBLISHED',
    featured: false,
    summary: 'A compact analytics widget concept.',
  },
];

function mockLabFetch(records: unknown, ok = true) {
  const fetchMock = vi.fn().mockResolvedValue({
    ok,
    status: ok ? 200 : 500,
    json: () => Promise.resolve(records),
  });
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

beforeEach(() => {
  vi.restoreAllMocks();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('WorkTabs', () => {
  it('selects Engineering Work by default without requesting Lab records', async () => {
    const fetchMock = mockLabFetch(labRecords);
    const user = userEvent.setup();

    render(
      <WorkTabs>
        <div>Engineering content</div>
      </WorkTabs>,
    );

    const engineeringTab = screen.getByRole('tab', { name: /engineering work/i });
    const labTab = screen.getByRole('tab', { name: /design lab/i });

    expect(engineeringTab).toHaveAttribute('aria-selected', 'true');
    expect(labTab).toHaveAttribute('aria-selected', 'false');
    expect(screen.getByText('Engineering content')).toBeInTheDocument();
    expect(screen.queryByText(/independent redesign concept/i)).not.toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalled();

    // Sanity: user interaction harness works before the switching test relies on it.
    await user.click(engineeringTab);
    expect(engineeringTab).toHaveAttribute('aria-selected', 'true');
  });

  it('switching to Design Lab requests published records, shows the disclaimer, and uses case-study actions', async () => {
    const fetchMock = mockLabFetch(labRecords);
    const user = userEvent.setup();

    render(
      <WorkTabs>
        <div>Engineering content</div>
      </WorkTabs>,
    );

    await user.click(screen.getByRole('tab', { name: /design lab/i }));

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0][0]).toContain('/api/portfolio/design-lab');

    const caseStudyLinks = await screen.findAllByRole('link', { name: /view case study/i });
    expect(caseStudyLinks).toHaveLength(2);

    // Required legal label is visible on the published Lab work.
    expect(screen.getAllByText(REQUIRED_DISCLAIMER).length).toBeGreaterThan(0);

    // Lab work must never be labeled as a GitHub engineering repository.
    expect(screen.queryByText(/view repository on github/i)).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: /github/i })).not.toBeInTheDocument();

    const exploreLink = screen.getByRole('link', { name: /explore design lab/i });
    expect(exploreLink).toHaveAttribute('href', '/lab');
  });

  it('routes Explore Design Lab and case-study fallbacks to the configured Lab origin', async () => {
    mockLabFetch(labRecords);
    const user = userEvent.setup();

    render(
      <WorkTabs labOrigin="https://lab.example.com">
        <div>Engineering content</div>
      </WorkTabs>,
    );

    await user.click(screen.getByRole('tab', { name: /design lab/i }));

    const caseStudyLinks = await screen.findAllByRole('link', { name: /view case study/i });
    // First record links to its published case-study URL; the record without one
    // falls back to the configured Lab origin plus slug.
    expect(caseStudyLinks[0]).toHaveAttribute('href', 'https://lab.example.com/checkout-redesign');
    expect(caseStudyLinks[1]).toHaveAttribute(
      'href',
      'https://lab.example.com/analytics-widget',
    );
    expect(screen.getByRole('link', { name: /explore design lab/i })).toHaveAttribute(
      'href',
      'https://lab.example.com',
    );
  });

  it('shows an error message when published Lab records cannot be loaded', async () => {
    mockLabFetch({ error: 'boom' }, false);
    const user = userEvent.setup();

    render(
      <WorkTabs>
        <div>Engineering content</div>
      </WorkTabs>,
    );

    await user.click(screen.getByRole('tab', { name: /design lab/i }));

    expect(await screen.findByRole('alert')).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: /view case study/i })).not.toBeInTheDocument();
  });

  it('keeps the approved heading and renders tabs around the layered coverflow', async () => {
    const fetchMock = mockLabFetch(labRecords);
    const user = userEvent.setup();

    render(<CoverflowSection projects={engineeringProjects} labProjects={labRecords} />);

    expect(
      screen.getByRole('heading', { name: /featured engineering work/i }),
    ).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /engineering work/i })).toHaveAttribute(
      'aria-selected',
      'true',
    );

    await user.click(screen.getByRole('tab', { name: /design lab/i }));

    // Injected records render without a network request.
    expect(fetchMock).not.toHaveBeenCalled();
    expect(await screen.findByText('Checkout Redesign')).toBeInTheDocument();
    expect(screen.getAllByRole('link', { name: /view case study/i })).toHaveLength(2);
  });
});
