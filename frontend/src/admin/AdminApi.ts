import type {
  AuthSession,
  OverviewResponse,
  ReviewQueueItem,
  ReviewDetailResponse,
  DecisionResponse,
  PublicationResponse,
  AuditSummary,
  ApiErrorPayload,
} from './types';

export class AdminApiError extends Error {
  status: number;
  code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.name = 'AdminApiError';
    this.status = status;
    this.code = code;
  }
}

function generateUUID(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

export class AdminApiClient {
  private csrfToken: string | null = null;
  private baseUrl: string;

  constructor(baseUrl: string = '') {
    this.baseUrl = baseUrl;
  }

  setCsrfToken(token: string | null) {
    this.csrfToken = token;
  }

  getCsrfToken(): string | null {
    return this.csrfToken;
  }

  private async ensureCsrfToken(): Promise<string> {
    if (this.csrfToken) {
      return this.csrfToken;
    }
    const session = await this.getSession();
    if (session.csrf_token) {
      this.csrfToken = session.csrf_token;
      return session.csrf_token;
    }
    return '';
  }

  private async mapError(res: Response): Promise<AdminApiError> {
    let payload: ApiErrorPayload | null = null;
    try {
      payload = await res.json();
    } catch {
      // Non-JSON error payload
    }

    const backendMsg = payload?.error;
    const code = payload?.code;

    let actionableMessage = backendMsg || res.statusText || 'An unexpected error occurred';
    switch (res.status) {
      case 400:
        actionableMessage = backendMsg || 'Invalid request parameters. Please verify input.';
        break;
      case 401:
        actionableMessage = 'Authentication required. Please sign in with an authorized account.';
        break;
      case 403:
        if (code === 'step_up_required') {
          actionableMessage = 'Step-up authentication required for high-privilege publishing.';
        } else {
          actionableMessage = backendMsg || 'Access forbidden. Owner permissions or valid CSRF token required.';
        }
        break;
      case 404:
        actionableMessage = backendMsg || 'Requested resource not found.';
        break;
      case 409:
        actionableMessage = backendMsg || 'Conflict: The submission state has changed or artifact hash mismatch.';
        break;
      case 500:
        actionableMessage = 'Internal server error. Please try again or inspect server logs.';
        break;
    }

    return new AdminApiError(actionableMessage, res.status, code);
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {},
    isMutation: boolean = false
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const headers: Record<string, string> = {
      ...(options.headers as Record<string, string>),
    };

    if (isMutation) {
      const csrf = await this.ensureCsrfToken();
      if (csrf) {
        headers['X-CSRF-Token'] = csrf;
      }
      if (!headers['X-Request-ID']) {
        headers['X-Request-ID'] = generateUUID();
      }
      if (!headers['Content-Type'] && options.body) {
        headers['Content-Type'] = 'application/json';
      }
    }

    const res = await fetch(url, {
      ...options,
      headers,
      credentials: 'same-origin',
    });

    if (!res.ok) {
      throw await this.mapError(res);
    }

    // Handle 204 No Content
    if (res.status === 204) {
      return {} as T;
    }

    return res.json() as Promise<T>;
  }

  async getSession(): Promise<AuthSession> {
    const res = await fetch(`${this.baseUrl}/api/admin/auth/session`, {
      method: 'GET',
      credentials: 'same-origin',
    });

    if (!res.ok) {
      throw await this.mapError(res);
    }

    const session: AuthSession = await res.json();
    if (session.csrf_token) {
      this.csrfToken = session.csrf_token;
    }
    return session;
  }

  async logout(): Promise<void> {
    await fetch(`${this.baseUrl}/api/admin/auth/logout`, {
      method: 'POST',
      credentials: 'same-origin',
    });
    this.csrfToken = null;
  }

  async getOverview(): Promise<OverviewResponse> {
    return this.request<OverviewResponse>('/api/admin/overview');
  }

  async getReviews(state?: string): Promise<ReviewQueueItem[]> {
    const query = state && state !== 'all' ? `?state=${encodeURIComponent(state)}` : '';
    return this.request<ReviewQueueItem[]>(`/api/admin/reviews${query}`);
  }

  async getReviewDetail(id: string): Promise<ReviewDetailResponse> {
    return this.request<ReviewDetailResponse>(`/api/admin/reviews/${encodeURIComponent(id)}`);
  }

  async requestChanges(id: string, reason: string): Promise<DecisionResponse> {
    return this.request<DecisionResponse>(
      `/api/admin/reviews/${encodeURIComponent(id)}/request-changes`,
      {
        method: 'POST',
        body: JSON.stringify({ reason }),
      },
      true
    );
  }

  async reject(id: string, reason: string): Promise<DecisionResponse> {
    return this.request<DecisionResponse>(
      `/api/admin/reviews/${encodeURIComponent(id)}/reject`,
      {
        method: 'POST',
        body: JSON.stringify({ reason }),
      },
      true
    );
  }

  async approveAndPublish(
    id: string,
    artifactSha256: string,
    idempotencyKey?: string
  ): Promise<PublicationResponse> {
    const idemKey = idempotencyKey || generateUUID();
    return this.request<PublicationResponse>(
      `/api/admin/reviews/${encodeURIComponent(id)}/approve-and-publish`,
      {
        method: 'POST',
        headers: {
          'Idempotency-Key': idemKey,
        },
        body: JSON.stringify({
          artifact_sha256: artifactSha256,
          idempotency_key: idemKey,
        }),
      },
      true
    );
  }

  async archiveProject(id: string): Promise<{ success: boolean; message: string }> {
    return this.request<{ success: boolean; message: string }>(
      `/api/admin/projects/${encodeURIComponent(id)}/archive`,
      {
        method: 'POST',
        body: JSON.stringify({}),
      },
      true
    );
  }

  async getAudit(limit: number = 50): Promise<AuditSummary[]> {
    return this.request<AuditSummary[]>(`/api/admin/audit?limit=${limit}`);
  }
}

export const adminApi = new AdminApiClient();
