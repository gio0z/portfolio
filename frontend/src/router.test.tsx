import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { AppRouter } from './router';
import { adminApi } from './admin/AdminApi';

describe('AppRouter', () => {
  beforeEach(() => {
    window.history.pushState({}, '', '/');
  });

  it('renders the admin shell only for /admin routes', () => {
    window.history.pushState({}, '', '/admin/reviews');
    render(<AppRouter />);
    expect(screen.getByRole('heading', { name: /review queue/i })).toBeInTheDocument();
    expect(screen.queryByText(/featured engineering work/i)).not.toBeInTheDocument();
  });

  it('renders the admin shell for /admin base route', () => {
    window.history.pushState({}, '', '/admin');
    render(<AppRouter />);
    expect(screen.getByRole('heading', { name: /review queue/i })).toBeInTheDocument();
    expect(screen.queryByText(/featured engineering work/i)).not.toBeInTheDocument();
  });

  it('renders the public portfolio shell for / routes', () => {
    window.history.pushState({}, '', '/');
    render(<AppRouter />);
    expect(screen.getByRole('heading', { name: /featured engineering work/i })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: /review queue/i })).not.toBeInTheDocument();
  });

  it('renders the lab shell for /lab routes', () => {
    window.history.pushState({}, '', '/lab');
    render(<AppRouter />);
    expect(screen.getByRole('heading', { name: /design lab/i })).toBeInTheDocument();
    expect(screen.queryByText(/featured engineering work/i)).not.toBeInTheDocument();
  });

  it('renders the lab shell for /lab/* nested routes', () => {
    window.history.pushState({}, '', '/lab/checkout-redesign');
    render(<AppRouter />);
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
    render(<AppRouter />);

    expect(await screen.findByText('Detail Route Project')).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /preview/i })).toBeInTheDocument();
  });
});
