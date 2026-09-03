export interface AuthSession {
  authenticated: boolean;
  login?: string;
  csrf_token?: string;
}

export interface OverviewResponse {
  pending_reviews: number;
  published_projects: number;
  build_failures: number;
  recent_audits: AuditSummary[];
}

export interface ReviewQueueItem {
  id: string;
  project_id: string;
  project_slug: string;
  project_title: string;
  original_product: string;
  revision: number;
  state: string;
  artifact_sha256: string;
  preview_url: string;
  build_result: string;
  test_result: string;
  security_scan_result: string;
  can_approve: boolean;
  submitted_by: string;
  submitted_at: string;
  updated_at: string;
  portfolio_metadata?: {
    tags?: string[];
    featured?: boolean;
    case_study_slug?: string;
    metrics?: Record<string, string>;
  };
}

export interface ArtifactDetail {
  sha256: string;
}

export interface ReviewDetailResponse {
  submission: unknown;
  project: unknown;
  artifact: ArtifactDetail;
  can_approve: boolean;
}

export interface RequestChangesInput {
  reason: string;
}

export interface RejectInput {
  reason: string;
}

export interface ApproveAndPublishInput {
  artifact_sha256: string;
  idempotency_key?: string;
}

export interface DecisionResponse {
  success: boolean;
  submission?: unknown;
}

export interface PublicationResponse {
  success: boolean;
  submission_id: string;
  revision: number;
  artifact_sha256: string;
  lab_url: string;
  portfolio_url: string;
  deployment_ids?: string[];
  destination_urls?: string[];
  published_at: string;
}

export interface AuditSummary {
  timestamp: string;
  request_id: string;
  actor_kind: string;
  actor_id: string;
  action: string;
  project_id?: string;
  submission_id?: string;
  result: string;
}

export interface ApiErrorPayload {
  error: string;
  code?: string;
}
