import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { IslandRouter } from './IslandRouter';
import { adminApi } from '../admin/AdminApi';

describe('IslandRouter', () => {
  beforeEach(() => {
    window.history.pushState({}, '', '/');
  });

  it('renders the admin shell only for /admin routes', () => {
    window.history.pushState({}, '', '/admin/reviews');
    render(<IslandRouter />);
    expect(screen.getByRole('heading', { name: /review queue/i })).toBeInTheDocument();
    expect(screen.queryByText(/featured engineering work/i)).not.toBeInTheDocument();
  });

  it('renders the admin shell for /admin base route', () => {
    window.history.pushState({}, '', '/admin');
    render(<IslandRouter />);
    expect(screen.getByRole('heading', { name: /review queue/i })).toBeInTheDocument();
    expect(screen.queryByText(/featured engineering work/i)).not.toBeInTheDocument();
  });

  it('leaves public routes to the prerendered HTML rather than routing them client-side', () => {
    // Static pages are served as finished documents, so the client router owns
    // no public route and must render none of its own surfaces here.
    window.history.pushState({}, '', '/');
    render(<IslandRouter />);
    expect(screen.queryByRole('heading', { name: /review queue/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: /design lab/i })).not.toBeInTheDocument();
  });

  it('renders the lab shell for /lab routes', () => {
    window.history.pushState({}, '', '/lab');
    render(<IslandRouter />);
    expect(screen.getByRole('heading', { name: /design lab/i })).toBeInTheDocument();
    expect(screen.queryByText(/featured engineering work/i)).not.toBeInTheDocument();
  });

  it('renders the lab shell for /lab/* nested routes', () => {
    window.history.pushState({}, '', '/lab/checkout-redesign');
    render(<IslandRouter />);
    expect(screen.getByRole('heading', { name: /design lab/i })).toBeInTheDocument();
    expect(screen.queryByText(/featured engineering work/i)).not.toBeInTheDocument();
  });

  it('routes /admin/reviews/:id to the review detail page', async () => {
    vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue({
      can_approve: false,
      artifact: { sha256: 'abc123' },
      submission: {
        id: 'sub-9',
        lab_project_id: 'proj-9',
        revision: 1,
        state: 'IN_REVIEW',
        artifact_sha256: 'abc123',
        preview_url: 'https://preview.local/x',
        build_result: 'passed',
        test_result: 'passed',
        security_scan_result: 'passed',
        submitted_by: 'agent',
        submitted_at: '2026-09-03T10:00:00Z',
        updated_at: '2026-09-03T10:30:00Z',
        portfolio_metadata: { summary: 'sum' },
      },
      project: {
        id: 'proj-9',
        slug: 'detail-slug',
        title: 'Detail Route Project',
        original_product: 'Acme',
        disclaimer: 'Independent redesign concept.',
        status: 'active',
        featured: false,
        created_at: '2026-09-01T08:00:00Z',
        updated_at: '2026-09-03T10:30:00Z',
      },
    });

    window.history.pushState({}, '', '/admin/reviews/sub-9');
    render(<IslandRouter />);

    expect(await screen.findByText('Detail Route Project')).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /preview/i })).toBeInTheDocument();
  });
});
