import { describe, it, expect, vi, beforeEach } from 'vitest';
import { AdminApiClient, AdminApiError } from './AdminApi';

describe('AdminApiClient', () => {
  let client: AdminApiClient;

  beforeEach(() => {
    vi.restoreAllMocks();
    client = new AdminApiClient('http://localhost:8080');
  });

  it('uses same-origin credentials for requests', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ pending_reviews: 1, published_projects: 5, build_failures: 0, recent_audits: [] }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    );

    await client.getOverview();

    expect(fetchSpy).toHaveBeenCalledWith(
      'http://localhost:8080/api/admin/overview',
      expect.objectContaining({
        credentials: 'same-origin',
      })
    );
  });

  it('reads and caches CSRF token on getSession and attaches to mutations', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch')
      // 1. Session call
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            authenticated: true,
            login: 'gio0z',
            csrf_token: 'test-csrf-token-12345',
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )
      // 2. Mutation call (request changes)
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ success: true }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      );

    const session = await client.getSession();
    expect(session.csrf_token).toBe('test-csrf-token-12345');
    expect(client.getCsrfToken()).toBe('test-csrf-token-12345');

    await client.requestChanges('sub-1', 'Need better error handling');

    expect(fetchSpy).toHaveBeenLastCalledWith(
      'http://localhost:8080/api/admin/reviews/sub-1/request-changes',
      expect.objectContaining({
        method: 'POST',
        credentials: 'same-origin',
        headers: expect.objectContaining({
          'X-CSRF-Token': 'test-csrf-token-12345',
          'X-Request-ID': expect.any(String),
          'Content-Type': 'application/json',
        }),
      })
    );
  });

  it('attaches X-Request-ID and Idempotency-Key on approveAndPublish', async () => {
    client.setCsrfToken('mock-csrf');
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(
        JSON.stringify({
          success: true,
          submission_id: 'sub-1',
          revision: 2,
          artifact_sha256: 'sha-value',
          lab_url: 'https://lab.example.com',
          portfolio_url: 'https://example.com',
          published_at: '2026-09-03T12:00:00Z',
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } }
      )
    );

    await client.approveAndPublish('sub-1', 'sha-value', 'custom-idempotency-key');

    expect(fetchSpy).toHaveBeenCalledWith(
      'http://localhost:8080/api/admin/reviews/sub-1/approve-and-publish',
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({
          'X-CSRF-Token': 'mock-csrf',
          'X-Request-ID': expect.any(String),
          'Idempotency-Key': 'custom-idempotency-key',
        }),
        body: JSON.stringify({
          artifact_sha256: 'sha-value',
          idempotency_key: 'custom-idempotency-key',
        }),
      })
    );
  });

  it('maps backend errors to actionable messages', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async () =>
      new Response(
        JSON.stringify({
          error: 'forbidden: step-up authentication required',
          code: 'step_up_required',
        }),
        { status: 403, headers: { 'Content-Type': 'application/json' } }
      )
    );

    await expect(client.getOverview()).rejects.toThrow(AdminApiError);
    try {
      await client.getOverview();
    } catch (err: unknown) {
      const apiErr = err as AdminApiError;
      expect(apiErr.status).toBe(403);
      expect(apiErr.code).toBe('step_up_required');
      expect(apiErr.message).toContain('Step-up authentication required');
    }
  });

  it('clears csrf token on logout', async () => {
    client.setCsrfToken('existing-token');
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ status: 'logged_out' }), { status: 200 })
    );

    await client.logout();
    expect(client.getCsrfToken()).toBeNull();
  });
});
