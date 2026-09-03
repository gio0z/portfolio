import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReviewQueue } from './ReviewQueue';
import { adminApi } from './AdminApi';
import type { ReviewQueueItem } from './types';

describe('ReviewQueue', () => {
  const mockSubmissions: ReviewQueueItem[] = [
    {
      id: 'sub-1',
      project_id: 'proj-1',
      project_slug: 'checkout-redesign',
      project_title: 'Checkout Redesign V2',
      original_product: 'checkout',
      revision: 1,
      state: 'IN_REVIEW',
      artifact_sha256: 'a1b2c3d4e5f678901234567890abcdef1234567890abcdef1234567890abcdef',
      preview_url: 'https://preview.example.com/checkout',
      build_result: 'passed',
      test_result: 'passed',
      security_scan_result: 'passed',
      can_approve: true,
      submitted_by: 'alice',
      submitted_at: '2026-09-03T10:00:00Z',
      updated_at: '2026-09-03T10:00:00Z',
    },
    {
      id: 'sub-2',
      project_id: 'proj-2',
      project_slug: 'analytics-widget',
      project_title: 'Analytics Dashboard Widget',
      original_product: 'analytics-hub',
      revision: 2,
      state: 'IN_REVIEW',
      artifact_sha256: 'f0e1d2c3b4a596877869504132abcdef0123456789abcdef0123456789abcdef',
      preview_url: 'https://preview.example.com/analytics',
      build_result: 'passed',
      test_result: 'passed',
      security_scan_result: 'failed',
      can_approve: false,
      submitted_by: 'bob',
      submitted_at: '2026-09-03T11:00:00Z',
      updated_at: '2026-09-03T11:00:00Z',
    },
  ];

  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('renders submissions and asserts draft title, original product, status, evidence, hash, and review actions', async () => {
    vi.spyOn(adminApi, 'getReviews').mockResolvedValue(mockSubmissions);

    render(<ReviewQueue />);

    // Wait for queue items to load
    await waitFor(() => {
      expect(screen.getByText('Checkout Redesign V2')).toBeInTheDocument();
    });

    // 1. Assert draft titles
    expect(screen.getByText('Checkout Redesign V2')).toBeInTheDocument();
    expect(screen.getByText('Analytics Dashboard Widget')).toBeInTheDocument();

    // 2. Assert original products
    expect(screen.getByText('checkout')).toBeInTheDocument();
    expect(screen.getByText('analytics-hub')).toBeInTheDocument();

    // 3. Assert status
    const statusBadges = screen.getAllByText(/in_review/i);
    expect(statusBadges.length).toBeGreaterThanOrEqual(2);

    // 4. Assert test/build/scan evidence
    expect(screen.getAllByText(/build: passed/i).length).toBeGreaterThanOrEqual(2);
    expect(screen.getAllByText(/test: passed/i).length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText(/scan: passed/i)).toBeInTheDocument();
    expect(screen.getByText(/scan: failed/i)).toBeInTheDocument();

    // 5. Assert artifact hashes
    expect(screen.getByText(/a1b2c3d4e5f6/i)).toBeInTheDocument();
    expect(screen.getByText(/f0e1d2c3b4a5/i)).toBeInTheDocument();

    // 6. Assert review actions appear
    const reviewButtons = screen.getAllByRole('button', { name: /review/i });
    expect(reviewButtons.length).toBeGreaterThanOrEqual(2);

    // 7. Assert a failed scan disables approval
    // Find row or card for sub-1 (can_approve: true, scan: passed)
    const row1 = screen.getByTestId('review-item-sub-1');
    const approveBtn1 = within(row1).getByRole('button', { name: /approve/i });
    expect(approveBtn1).toBeEnabled();

    // Find row or card for sub-2 (can_approve: false, scan: failed)
    const row2 = screen.getByTestId('review-item-sub-2');
    const approveBtn2 = within(row2).getByRole('button', { name: /approve/i });
    expect(approveBtn2).toBeDisabled();
  });

  it('handles approve-and-publish action for eligible submission', async () => {
    vi.spyOn(adminApi, 'getReviews').mockResolvedValue(mockSubmissions);
    const approveSpy = vi.spyOn(adminApi, 'approveAndPublish').mockResolvedValue({
      success: true,
      submission_id: 'sub-1',
      revision: 1,
      artifact_sha256: mockSubmissions[0].artifact_sha256,
      lab_url: 'https://lab.example.com/checkout',
      portfolio_url: 'https://example.com/projects/checkout',
      published_at: new Date().toISOString(),
    });

    const user = userEvent.setup();
    render(<ReviewQueue />);

    await waitFor(() => {
      expect(screen.getByText('Checkout Redesign V2')).toBeInTheDocument();
    });

    const row1 = screen.getByTestId('review-item-sub-1');
    const approveBtn1 = within(row1).getByRole('button', { name: /approve/i });
    await user.click(approveBtn1);

    expect(approveSpy).toHaveBeenCalledWith('sub-1', mockSubmissions[0].artifact_sha256);
  });

  it('handles empty queue gracefully', async () => {
    vi.spyOn(adminApi, 'getReviews').mockResolvedValue([]);

    render(<ReviewQueue />);

    await waitFor(() => {
      expect(screen.getByText(/no submissions in review queue/i)).toBeInTheDocument();
    });
  });
});
