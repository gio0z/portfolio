import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { ReviewDetail } from './ReviewDetail';
import { adminApi } from './AdminApi';
import type { ReviewDetailResponse } from './types';

describe('ReviewDetail and Approval UI', () => {
  const mockApprovedDetail: ReviewDetailResponse = {
    can_approve: true,
    artifact: {
      sha256: '9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08',
    },
    submission: {
      id: 'sub-100',
      lab_project_id: 'proj-50',
      revision: 1,
      state: 'IN_REVIEW',
      artifact_sha256: '9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08',
      preview_url: 'https://preview.internal.local/checkout-redesign',
      build_result: 'passed',
      test_result: 'passed',
      security_scan_result: 'passed',
      submitted_by: 'design-agent-42',
      submitted_at: '2026-09-03T10:00:00Z',
      updated_at: '2026-09-03T10:30:00Z',
      portfolio_metadata: {
        summary: 'Streamlined multi-step checkout to single-page flow',
        description: 'Complete redesign focusing on mobile conversion and reduced cognitive load.',
        tags: ['React', 'Checkout', 'UX Optimization'],
        featured: true,
        case_study_slug: 'checkout-v2',
        metrics: {
          conversion_lift: '+18.4%',
          latency_drop: '-240ms',
        },
        before_image_url: 'https://cdn.local/assets/checkout-before.png',
        after_image_url: 'https://cdn.local/assets/checkout-after.png',
        build_logs: 'Build completed in 3.4s without warnings.',
        test_logs: '14/14 tests passed.',
        security_scan_logs: 'Static analysis clean. 0 vulnerabilities found.',
        assets: [
          { name: 'checkout-before.png', url: 'https://cdn.local/assets/checkout-before.png', size: 102400 },
          { name: 'checkout-after.png', url: 'https://cdn.local/assets/checkout-after.png', size: 98304 },
        ],
      },
    },
    project: {
      id: 'proj-50',
      slug: 'checkout-flow',
      title: 'Checkout Flow Redesign',
      original_product: 'Stripe Elements Checkout',
      disclaimer: 'Independent exploration not affiliated with or endorsed by Stripe.',
      focus: ['UX Architecture', 'Checkout Friction', 'Mobile First'],
      platforms: ['Web', 'Mobile Safari'],
      status: 'active',
      featured: true,
      created_at: '2026-09-01T08:00:00Z',
      updated_at: '2026-09-03T10:30:00Z',
    },
  };

  const mockFailedDetail: ReviewDetailResponse = {
    ...mockApprovedDetail,
    can_approve: false,
    submission: {
      ...mockApprovedDetail.submission,
      security_scan_result: 'failed',
      portfolio_metadata: {
        ...mockApprovedDetail.submission.portfolio_metadata,
        security_scan_logs: 'CRITICAL: Dangerous eval detected in bundle.',
      },
    },
  };

  const mockSourceDetail: ReviewDetailResponse = {
    ...mockApprovedDetail,
    project: {
      ...mockApprovedDetail.project,
      source_urls: ['https://www.stripe.com/payments', 'https://stripe.com/docs/checkout'],
    },
  };

  beforeEach(() => {
    vi.restoreAllMocks();
  });

  function renderWithRouter(reviewId: string = 'sub-100') {
    return render(
      <MemoryRouter initialEntries={[`/admin/reviews/${reviewId}`]}>
        <Routes>
          <Route path="/admin/reviews/:id" element={<ReviewDetail />} />
        </Routes>
      </MemoryRouter>
    );
  }

  describe('Security and Sandbox', () => {
    it('renders preview iframe with strict sandbox without allow-same-origin', async () => {
      vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue(mockApprovedDetail);

      renderWithRouter();

      await waitFor(() => {
        expect(screen.getByText('Checkout Flow Redesign')).toBeInTheDocument();
      });

      const iframe = screen.getByTestId('preview-iframe') as HTMLIFrameElement;
      expect(iframe).toBeInTheDocument();
      expect(iframe.getAttribute('src')).toBe('https://preview.internal.local/checkout-redesign');

      const sandboxAttr = iframe.getAttribute('sandbox') || '';
      expect(sandboxAttr).toContain('allow-scripts');
      expect(sandboxAttr).toContain('allow-forms');
      expect(sandboxAttr).not.toContain('allow-same-origin');
    });

    it('supports switching between desktop and mobile viewport dimensions', async () => {
      vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue(mockApprovedDetail);
      const user = userEvent.setup();

      renderWithRouter();

      await waitFor(() => {
        expect(screen.getByTestId('preview-iframe')).toBeInTheDocument();
      });

      const container = screen.getByTestId('preview-container');
      const mobileBtn = screen.getByRole('button', { name: /mobile/i });
      const desktopBtn = screen.getByRole('button', { name: /desktop/i });

      // Default desktop view
      expect(container.className).toContain('desktop');

      // Switch to mobile
      await user.click(mobileBtn);
      expect(container.className).toContain('mobile');

      // Switch back to desktop
      await user.click(desktopBtn);
      expect(container.className).toContain('desktop');
    });
  });

  describe('Review Tabs and Evidence', () => {
    it('displays prominent artifact SHA-256 across views', async () => {
      vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue(mockApprovedDetail);

      renderWithRouter();

      await waitFor(() => {
        expect(screen.getByText('Checkout Flow Redesign')).toBeInTheDocument();
      });

      // SHA-256 should be displayed prominently
      const shaElements = screen.getAllByText(/9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08/i);
      expect(shaElements.length).toBeGreaterThanOrEqual(1);
    });

    it('renders Before/After tab with comparison assets', async () => {
      vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue(mockApprovedDetail);
      const user = userEvent.setup();

      renderWithRouter();

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /before\/after/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('tab', { name: /before\/after/i }));

      expect(screen.getByAltText(/before redesign/i)).toHaveAttribute('src', 'https://cdn.local/assets/checkout-before.png');
      expect(screen.getByAltText(/after redesign/i)).toHaveAttribute('src', 'https://cdn.local/assets/checkout-after.png');
    });

    it('renders Evidence tab with build, test, and security scan status and logs', async () => {
      vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue(mockApprovedDetail);
      const user = userEvent.setup();

      renderWithRouter();

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /evidence/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('tab', { name: /evidence/i }));

      expect(screen.getByTestId('evidence-build-status')).toHaveTextContent(/passed/i);
      expect(screen.getByTestId('evidence-test-status')).toHaveTextContent(/passed/i);
      expect(screen.getByTestId('evidence-scan-status')).toHaveTextContent(/passed/i);

      expect(screen.getByText(/Build completed in 3.4s without warnings/i)).toBeInTheDocument();
      expect(screen.getByText(/14\/14 tests passed/i)).toBeInTheDocument();
      expect(screen.getByText(/Static analysis clean/i)).toBeInTheDocument();
    });

    it('renders Portfolio Card tab with public metadata representation', async () => {
      vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue(mockApprovedDetail);
      const user = userEvent.setup();

      renderWithRouter();

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /portfolio card/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('tab', { name: /portfolio card/i }));

      expect(screen.getByText('Streamlined multi-step checkout to single-page flow')).toBeInTheDocument();
      expect(screen.getByText('+18.4%')).toBeInTheDocument();
      expect(screen.getByText('-240ms')).toBeInTheDocument();
      expect(screen.getByText(/React/i)).toBeInTheDocument();
    });

    it('renders Lab Page tab with project details and disclaimer', async () => {
      vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue(mockApprovedDetail);
      const user = userEvent.setup();

      renderWithRouter();

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /lab page/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('tab', { name: /lab page/i }));

      const labPanel = screen.getByRole('tabpanel');
      expect(within(labPanel).getByText(/checkout-flow/i)).toBeInTheDocument();
      expect(within(labPanel).getByText(/Stripe Elements Checkout/i)).toBeInTheDocument();
      expect(
        within(labPanel).getByText(
          /Independent exploration not affiliated with or endorsed by Stripe/i
        )
      ).toBeInTheDocument();
    });
  });

  describe('Source-site Reference Links and Origin Verdict', () => {
    it('renders source-site reference links as safe outbound links with the raw URL as text', async () => {
      vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue(mockSourceDetail);

      renderWithRouter();

      await waitFor(() => {
        expect(screen.getByTestId('source-urls-block')).toBeInTheDocument();
      });

      const block = screen.getByTestId('source-urls-block');
      const links = within(block).getAllByRole('link');
      expect(links).toHaveLength(2);

      const firstLink = within(block).getByRole('link', {
        name: /https:\/\/www\.stripe\.com\/payments/i,
      });
      expect(firstLink).toHaveAttribute('href', 'https://www.stripe.com/payments');
      expect(firstLink).toHaveAttribute('target', '_blank');
      expect(firstLink).toHaveAttribute('rel', 'noopener noreferrer');

      const secondLink = within(block).getByRole('link', {
        name: /https:\/\/stripe\.com\/docs\/checkout/i,
      });
      expect(secondLink).toHaveAttribute('href', 'https://stripe.com/docs/checkout');
      expect(secondLink).toHaveAttribute('target', '_blank');
      expect(secondLink).toHaveAttribute('rel', 'noopener noreferrer');
    });

    it('omits the source-site block entirely when the project has no source URLs', async () => {
      vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue(mockApprovedDetail);

      renderWithRouter();

      await waitFor(() => {
        expect(screen.getByText('Checkout Flow Redesign')).toBeInTheDocument();
      });

      expect(screen.queryByTestId('source-urls-block')).not.toBeInTheDocument();
      expect(screen.queryByTestId('origin-verdict')).not.toBeInTheDocument();
      expect(
        screen.queryByRole('heading', { name: /source-site reference links/i })
      ).not.toBeInTheDocument();
    });

    it('omits the source-site block when the project source URLs list is empty', async () => {
      vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue({
        ...mockApprovedDetail,
        project: { ...mockApprovedDetail.project, source_urls: [] },
      });

      renderWithRouter();

      await waitFor(() => {
        expect(screen.getByText('Checkout Flow Redesign')).toBeInTheDocument();
      });

      expect(screen.queryByTestId('source-urls-block')).not.toBeInTheDocument();
    });

    it('renders the origin-vs-redesign verdict when source URLs exist', async () => {
      vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue(mockSourceDetail);

      renderWithRouter();

      await waitFor(() => {
        expect(screen.getByTestId('origin-verdict')).toBeInTheDocument();
      });

      const verdict = screen.getByTestId('origin-verdict');
      expect(verdict).toHaveTextContent(/original third-party products/i);
      expect(verdict).toHaveTextContent(/independent redesign concept/i);
      expect(verdict).toHaveTextContent(/not affiliated with/i);
      expect(verdict).toHaveTextContent(/endorsed by the original company/i);
    });
  });

  describe('Approval Decisions and Step-Up Flow', () => {
    it('disables Approve & Publish when security scan or build/test fails', async () => {
      vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue(mockFailedDetail);

      renderWithRouter();

      await waitFor(() => {
        expect(screen.getByText('Checkout Flow Redesign')).toBeInTheDocument();
      });

      const approveBtn = screen.getByRole('button', { name: /approve & publish/i });
      expect(approveBtn).toBeDisabled();
    });

    it('requires step-up confirmation and summarizes dual publication destinations in ApprovalDialog', async () => {
      vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue(mockApprovedDetail);
      const approveSpy = vi.spyOn(adminApi, 'approveAndPublish').mockResolvedValue({
        success: true,
        submission_id: 'sub-100',
        revision: 1,
        artifact_sha256: mockApprovedDetail.artifact.sha256,
        lab_url: 'https://lab.example.com/checkout-flow',
        portfolio_url: 'https://example.com/projects/checkout-flow',
        published_at: '2026-09-03T11:00:00Z',
      });

      const user = userEvent.setup();
      renderWithRouter();

      await waitFor(() => {
        expect(screen.getByText('Checkout Flow Redesign')).toBeInTheDocument();
      });

      const approveBtn = screen.getByRole('button', { name: /approve & publish/i });
      expect(approveBtn).toBeEnabled();
      await user.click(approveBtn);

      // Modal appears
      const dialog = screen.getByRole('dialog', { name: /approve & publish/i });
      expect(dialog).toBeInTheDocument();

      // Summarizes atomic dual-destination publication
      const destinations = within(dialog).getByTestId('publication-destinations');
      expect(within(destinations).getByText('Design Lab', { selector: 'strong' })).toBeInTheDocument();
      expect(within(destinations).getByText('Portfolio', { selector: 'strong' })).toBeInTheDocument();
      expect(dialog).toHaveTextContent(/dual-destination/i);

      // Displays exact reviewed artifact hash
      expect(within(dialog).getByText(mockApprovedDetail.artifact.sha256)).toBeInTheDocument();

      // Confirm button is disabled before step-up input
      const confirmPublishBtn = within(dialog).getByRole('button', { name: /confirm & publish|confirm publication/i });
      expect(confirmPublishBtn).toBeDisabled();

      // Fill in step-up verification confirmation
      const stepUpInput = within(dialog).getByTestId('step-up-input');
      await user.type(stepUpInput, 'CONFIRM');

      expect(confirmPublishBtn).toBeEnabled();
      await user.click(confirmPublishBtn);

      // Verify AdminApi.approveAndPublish was invoked with artifact_sha256 and idempotency_key
      expect(approveSpy).toHaveBeenCalledWith(
        'sub-100',
        mockApprovedDetail.artifact.sha256,
        expect.any(String)
      );

      // Check success notification
      await waitFor(() => {
        expect(screen.getByText(/published successfully/i)).toBeInTheDocument();
      });
    });

    it('handles Request Changes decision', async () => {
      vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue(mockApprovedDetail);
      const requestChangesSpy = vi.spyOn(adminApi, 'requestChanges').mockResolvedValue({
        success: true,
      });
      vi.spyOn(window, 'prompt').mockReturnValue('Please fix typography alignment');

      const user = userEvent.setup();
      renderWithRouter();

      await waitFor(() => {
        expect(screen.getByText('Checkout Flow Redesign')).toBeInTheDocument();
      });

      const requestChangesBtn = screen.getByRole('button', { name: /request changes/i });
      await user.click(requestChangesBtn);

      expect(requestChangesSpy).toHaveBeenCalledWith('sub-100', 'Please fix typography alignment');
      await waitFor(() => {
        expect(screen.getByText(/changes requested/i)).toBeInTheDocument();
      });
    });

    it('handles Reject decision', async () => {
      vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue(mockApprovedDetail);
      const rejectSpy = vi.spyOn(adminApi, 'reject').mockResolvedValue({
        success: true,
      });
      vi.spyOn(window, 'prompt').mockReturnValue('Does not meet quality standards');

      const user = userEvent.setup();
      renderWithRouter();

      await waitFor(() => {
        expect(screen.getByText('Checkout Flow Redesign')).toBeInTheDocument();
      });

      const rejectBtn = screen.getByRole('button', { name: /reject/i });
      await user.click(rejectBtn);

      expect(rejectSpy).toHaveBeenCalledWith('sub-100', 'Does not meet quality standards');
      await waitFor(() => {
        expect(screen.getByText(/submission rejected/i)).toBeInTheDocument();
      });
    });

    it('handles Archive Project decision', async () => {
      vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue(mockApprovedDetail);
      const archiveSpy = vi.spyOn(adminApi, 'archiveProject').mockResolvedValue({
        success: true,
        message: 'Project archived successfully',
      });
      vi.spyOn(window, 'confirm').mockReturnValue(true);

      const user = userEvent.setup();
      renderWithRouter();

      await waitFor(() => {
        expect(screen.getByText('Checkout Flow Redesign')).toBeInTheDocument();
      });

      const archiveBtn = screen.getByRole('button', { name: /archive project/i });
      await user.click(archiveBtn);

      expect(archiveSpy).toHaveBeenCalledWith('proj-50');
      await waitFor(() => {
        expect(screen.getByText(/project archived/i)).toBeInTheDocument();
      });
    });

    it('requires step-up confirmation again after the dialog is cancelled and reopened', async () => {
      vi.spyOn(adminApi, 'getReviewDetail').mockResolvedValue(mockApprovedDetail);
      const approveSpy = vi.spyOn(adminApi, 'approveAndPublish').mockResolvedValue({
        success: true,
        submission_id: 'sub-100',
        revision: 1,
        artifact_sha256: mockApprovedDetail.artifact.sha256,
        lab_url: 'https://lab.example.com/checkout-flow',
        portfolio_url: 'https://example.com/projects/checkout-flow',
        published_at: '2026-09-03T11:00:00Z',
      });

      const user = userEvent.setup();
      renderWithRouter();

      await waitFor(() => {
        expect(screen.getByText('Checkout Flow Redesign')).toBeInTheDocument();
      });

      await user.click(screen.getByRole('button', { name: /approve & publish/i }));
      await user.type(screen.getByTestId('step-up-input'), 'CONFIRM');
      await user.click(screen.getByRole('button', { name: /cancel/i }));
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();

      await user.click(screen.getByRole('button', { name: /approve & publish/i }));
      const dialog = screen.getByRole('dialog');
      expect(within(dialog).getByTestId('step-up-input')).toHaveValue('');
      expect(within(dialog).getByRole('button', { name: /confirm & publish/i })).toBeDisabled();
      expect(approveSpy).not.toHaveBeenCalled();
    });
  });
});
