import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AdminOverview } from './AdminOverview';
import { adminApi } from './AdminApi';
import type { OverviewResponse } from './types';

describe('AdminOverview', () => {
  const mockOverviewData: OverviewResponse = {
    pending_reviews: 4,
    published_projects: 15,
    build_failures: 2,
    recent_audits: [
      {
        timestamp: '2026-09-03T12:30:00Z',
        request_id: 'req-1',
        actor_kind: 'OWNER',
        actor_id: 'gio0z',
        action: 'approve_and_publish',
        project_id: 'proj-1',
        submission_id: 'sub-1',
        result: 'success',
      },
      {
        timestamp: '2026-09-03T11:15:00Z',
        request_id: 'req-2',
        actor_kind: 'MCP_AGENT',
        actor_id: 'agent-codex',
        action: 'submit_revision',
        project_id: 'proj-2',
        submission_id: 'sub-2',
        result: 'success',
      },
    ],
  };

  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('renders overview metrics: pending reviews, build failures, published projects, latest deployment, and recent activity', async () => {
    vi.spyOn(adminApi, 'getOverview').mockResolvedValue(mockOverviewData);

    render(
      <MemoryRouter>
        <AdminOverview />
      </MemoryRouter>
    );

    // Wait for metrics to load
    await waitFor(() => {
      expect(screen.getByText('4')).toBeInTheDocument();
    });

    // 1. Pending reviews count
    expect(screen.getByText(/pending reviews/i)).toBeInTheDocument();
    expect(screen.getByText('4')).toBeInTheDocument();

    // 2. Published projects count
    expect(screen.getByText(/published projects/i)).toBeInTheDocument();
    expect(screen.getByText('15')).toBeInTheDocument();

    // 3. Build failures count
    expect(screen.getByText(/build failures/i)).toBeInTheDocument();
    expect(screen.getByText('2')).toBeInTheDocument();

    // 4. Latest deployment
    expect(screen.getByText(/latest deployment/i)).toBeInTheDocument();
    expect(screen.getAllByText(/approve_and_publish/i).length).toBeGreaterThanOrEqual(1);

    // 5. Recent activity
    expect(screen.getByText(/recent audit & mcp activity/i)).toBeInTheDocument();
    expect(screen.getByText(/agent-codex/i)).toBeInTheDocument();
    expect(screen.getByText(/submit_revision/i)).toBeInTheDocument();
  });

  it('handles error state gracefully', async () => {
    vi.spyOn(adminApi, 'getOverview').mockRejectedValue(new Error('Network error'));

    render(
      <MemoryRouter>
        <AdminOverview />
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByRole('alert')).toBeInTheDocument();
    });
  });
});
