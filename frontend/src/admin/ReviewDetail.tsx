import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import {
  ArrowLeft,
  CheckCircle,
  AlertTriangle,
  Send,
  Ban,
  Archive,
  ExternalLink,
  Clock,
  Sparkles,
} from 'lucide-react';
import { adminApi, AdminApiError } from './AdminApi';
import type { ReviewDetailResponse } from './types';
import { PreviewFrame } from './PreviewFrame';
import { EvidencePanel } from './EvidencePanel';
import { ApprovalDialog } from './ApprovalDialog';

type TabKey = 'preview' | 'before_after' | 'evidence' | 'portfolio_card' | 'lab_page';

export function ReviewDetail() {
  const { id } = useParams<{ id: string }>();
  const [detail, setDetail] = useState<ReviewDetailResponse | null>(null);
  const [loadedId, setLoadedId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [retryNonce, setRetryNonce] = useState<number>(0);
  const [activeTab, setActiveTab] = useState<TabKey>('preview');
  const [actionSuccess, setActionSuccess] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [isApprovalOpen, setIsApprovalOpen] = useState<boolean>(false);
  const [isProcessing, setIsProcessing] = useState<boolean>(false);

  // Silently refetch the current review without toggling the full-page loader.
  // Keeps the reviewed artifact on screen; surfaces refresh failures inline.
  const refreshDetail = async (reviewId: string) => {
    try {
      const data = await adminApi.getReviewDetail(reviewId);
      setDetail(data);
    } catch (err) {
      setActionError(
        err instanceof AdminApiError ? err.message : 'Failed to refresh review detail.'
      );
    }
  };

  useEffect(() => {
    if (!id) return;
    let active = true;
    adminApi
      .getReviewDetail(id)
      .then((data) => {
        if (!active) return;
        setDetail(data);
        setLoadedId(id);
        setError(null);
      })
      .catch((err) => {
        if (!active) return;
        setError(err instanceof AdminApiError ? err.message : 'Failed to load review detail.');
        setLoadedId(id);
      });
    return () => {
      active = false;
    };
  }, [id, retryNonce]);

  const handleRetry = () => {
    setError(null);
    setLoadedId(null);
    setRetryNonce((n) => n + 1);
  };

  const loading = loadedId !== id;

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center p-12 text-zinc-500">
        <div className="w-8 h-8 border-4 border-zinc-200 border-t-zinc-800 rounded-full animate-spin mb-4" />
        <p className="text-sm font-medium">Loading review details...</p>
      </div>
    );
  }

  if (error || !detail) {
    return (
      <div className="p-8 max-w-4xl mx-auto">
        <div className="bg-red-50 border border-red-200 rounded-xl p-6 text-center">
          <AlertTriangle className="w-8 h-8 text-red-600 mx-auto mb-2" />
          <h3 className="text-base font-bold text-red-900">Unable to load review</h3>
          <p className="text-sm text-red-700 mt-1">{error || 'Review not found.'}</p>
          <div className="mt-4 flex justify-center gap-3">
            <Link
              to="/admin/reviews"
              className="px-4 py-2 text-xs font-medium text-zinc-700 bg-white border border-zinc-300 rounded-lg hover:bg-zinc-50"
            >
              Back to Queue
            </Link>
            {id && (
              <button
                type="button"
                onClick={handleRetry}
                className="px-4 py-2 text-xs font-medium text-white bg-red-600 rounded-lg hover:bg-red-700"
              >
                Retry
              </button>
            )}
          </div>
        </div>
      </div>
    );
  }

  const { submission, project, artifact, can_approve } = detail;
  const meta = submission.portfolio_metadata || {};
  const sourceUrls = project.source_urls ?? [];

  const isBuildPass = submission.build_result?.toLowerCase() === 'passed';
  const isTestPass = submission.test_result?.toLowerCase() === 'passed';
  const isScanPass = submission.security_scan_result?.toLowerCase() === 'passed';
  const hasSha = Boolean(artifact?.sha256 && artifact.sha256.trim());

  // Can approve only if all checks passed, SHA is present, and submission state allows it
  const isEligibleForApproval = can_approve && isBuildPass && isTestPass && isScanPass && hasSha;

  const handleApproveConfirm = async () => {
    if (!id || !hasSha) return;
    setIsProcessing(true);
    setActionError(null);
    try {
      const idempotencyKey = crypto.randomUUID?.() || `pub-${Date.now()}`;
      await adminApi.approveAndPublish(id, artifact.sha256, idempotencyKey);
      setActionSuccess(`Project "${project.title}" published successfully to Design Lab and Portfolio!`);
      setIsApprovalOpen(false);
      await refreshDetail(id);
    } catch (err) {
      if (err instanceof AdminApiError) {
        setActionError(err.message);
      } else {
        setActionError('Failed to approve and publish project.');
      }
    } finally {
      setIsProcessing(false);
    }
  };

  const handleRequestChanges = async () => {
    if (!id) return;
    const reason = window.prompt('Enter reason for requesting changes:');
    if (!reason || !reason.trim()) return;

    setIsProcessing(true);
    setActionError(null);
    setActionSuccess(null);
    try {
      await adminApi.requestChanges(id, reason.trim());
      setActionSuccess(`Changes requested for "${project.title}".`);
      await refreshDetail(id);
    } catch (err) {
      if (err instanceof AdminApiError) {
        setActionError(err.message);
      } else {
        setActionError('Failed to request changes.');
      }
    } finally {
      setIsProcessing(false);
    }
  };

  const handleReject = async () => {
    if (!id) return;
    const reason = window.prompt('Enter reason for rejection:');
    if (!reason || !reason.trim()) return;

    setIsProcessing(true);
    setActionError(null);
    setActionSuccess(null);
    try {
      await adminApi.reject(id, reason.trim());
      setActionSuccess(`Submission rejected.`);
      await refreshDetail(id);
    } catch (err) {
      if (err instanceof AdminApiError) {
        setActionError(err.message);
      } else {
        setActionError('Failed to reject submission.');
      }
    } finally {
      setIsProcessing(false);
    }
  };

  const handleArchive = async () => {
    if (!project?.id) return;
    const confirmed = window.confirm(
      `Are you sure you want to archive project "${project.title}"? This will hide it from active public listings.`
    );
    if (!confirmed) return;

    setIsProcessing(true);
    setActionError(null);
    setActionSuccess(null);
    try {
      await adminApi.archiveProject(project.id);
      setActionSuccess(`Project archived successfully.`);
      if (id) {
        await refreshDetail(id);
      }
    } catch (err) {
      if (err instanceof AdminApiError) {
        setActionError(err.message);
      } else {
        setActionError('Failed to archive project.');
      }
    } finally {
      setIsProcessing(false);
    }
  };

  return (
    <div className="space-y-6 max-w-7xl mx-auto pb-12">
      {/* Top Bar with back link & notification messages */}
      <div className="flex items-center justify-between">
        <Link
          to="/admin/reviews"
          className="inline-flex items-center space-x-1.5 text-xs font-semibold text-zinc-600 hover:text-zinc-900 transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          <span>Back to Review Queue</span>
        </Link>
        <div className="flex items-center space-x-2 text-xs text-zinc-500 font-mono">
          <span>Submission ID:</span>
          <span className="text-zinc-800 font-semibold">{submission.id}</span>
        </div>
      </div>

      {/* Success Notification */}
      {actionSuccess && (
        <div className="bg-emerald-50 border border-emerald-200 text-emerald-800 px-4 py-3 rounded-xl text-sm flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <CheckCircle className="w-4 h-4 text-emerald-600 shrink-0" />
            <span>{actionSuccess}</span>
          </div>
          <button
            type="button"
            onClick={() => setActionSuccess(null)}
            className="text-emerald-600 hover:text-emerald-800 text-xs font-semibold"
          >
            Dismiss
          </button>
        </div>
      )}

      {/* Error Notification */}
      {actionError && (
        <div className="bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-xl text-sm flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <AlertTriangle className="w-4 h-4 text-red-600 shrink-0" />
            <span>{actionError}</span>
          </div>
          <button
            type="button"
            onClick={() => setActionError(null)}
            className="text-red-600 hover:text-red-800 text-xs font-semibold"
          >
            Dismiss
          </button>
        </div>
      )}

      {/* Header Card with Meta & Action Bar */}
      <div className="bg-white rounded-2xl border border-zinc-200 p-6 shadow-sm">
        <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-6">
          <div className="space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <span className="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-blue-50 text-blue-700 border border-blue-200">
                {project.original_product || 'Design Lab Redesign'}
              </span>
              <span className="px-2.5 py-0.5 rounded-full text-xs font-medium bg-zinc-100 text-zinc-700 border border-zinc-200">
                Revision {submission.revision}
              </span>
              <span
                className={`px-2.5 py-0.5 rounded-full text-xs font-semibold uppercase ${
                  submission.state === 'IN_REVIEW'
                    ? 'bg-amber-50 text-amber-700 border border-amber-200'
                    : submission.state === 'APPROVED' || submission.state === 'PUBLISHED'
                    ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
                    : 'bg-zinc-100 text-zinc-600 border border-zinc-200'
                }`}
              >
                {submission.state}
              </span>
            </div>
            <h1 className="text-2xl font-black tracking-tight text-zinc-900">
              {project.title}
            </h1>
            <p className="text-xs text-zinc-500 flex items-center gap-2">
              <span>Submitted by: <strong className="text-zinc-700">{submission.submitted_by}</strong></span>
              <span>•</span>
              <Clock className="w-3.5 h-3.5 inline text-zinc-400" />
              <span>{new Date(submission.submitted_at).toLocaleString()}</span>
            </p>
          </div>

          {/* Action Buttons Bar */}
          <div className="flex flex-wrap items-center gap-2.5">
            <button
              type="button"
              onClick={handleRequestChanges}
              disabled={isProcessing}
              className="flex items-center space-x-1.5 px-3.5 py-2 text-xs font-semibold text-amber-800 bg-amber-50 border border-amber-300 rounded-lg hover:bg-amber-100 transition-colors disabled:opacity-50"
            >
              <Send className="w-3.5 h-3.5" />
              <span>Request Changes</span>
            </button>

            <button
              type="button"
              onClick={handleReject}
              disabled={isProcessing}
              className="flex items-center space-x-1.5 px-3.5 py-2 text-xs font-semibold text-red-800 bg-red-50 border border-red-300 rounded-lg hover:bg-red-100 transition-colors disabled:opacity-50"
            >
              <Ban className="w-3.5 h-3.5" />
              <span>Reject</span>
            </button>

            <button
              type="button"
              onClick={handleArchive}
              disabled={isProcessing}
              className="flex items-center space-x-1.5 px-3.5 py-2 text-xs font-semibold text-zinc-700 bg-white border border-zinc-300 rounded-lg hover:bg-zinc-50 transition-colors disabled:opacity-50"
            >
              <Archive className="w-3.5 h-3.5" />
              <span>Archive Project</span>
            </button>

            <button
              type="button"
              onClick={() => setIsApprovalOpen(true)}
              disabled={!isEligibleForApproval || isProcessing}
              title={
                !isEligibleForApproval
                  ? 'Approval blocked: build, test, and security scan must pass with valid artifact hash.'
                  : 'Approve and publish atomically to Lab and Portfolio'
              }
              className={`flex items-center space-x-1.5 px-4 py-2 text-xs font-bold rounded-lg transition-all shadow-sm ${
                isEligibleForApproval && !isProcessing
                  ? 'bg-emerald-600 text-white hover:bg-emerald-700 hover:shadow'
                  : 'bg-zinc-200 text-zinc-400 cursor-not-allowed'
              }`}
            >
              <CheckCircle className="w-4 h-4" />
              <span>Approve &amp; Publish</span>
            </button>
          </div>
        </div>

        {/* SHA-256 Prominent Bar */}
        <div className="mt-5 pt-4 border-t border-zinc-100 flex flex-col sm:flex-row sm:items-center justify-between gap-2 text-xs font-mono bg-zinc-50 p-3 rounded-xl border border-zinc-200">
          <div className="flex items-center space-x-2 text-zinc-600">
            <span className="font-semibold text-zinc-500 uppercase tracking-wider text-[10px]">Artifact SHA256:</span>
            <span className="font-bold text-zinc-900 select-all break-all">{artifact?.sha256 || 'None'}</span>
          </div>
          <div className="flex items-center space-x-2 shrink-0">
            <span
              className={`px-2 py-0.5 rounded text-[11px] font-semibold ${
                isScanPass ? 'bg-emerald-100 text-emerald-800' : 'bg-red-100 text-red-800'
              }`}
            >
              Scan: {submission.security_scan_result}
            </span>
            <span
              className={`px-2 py-0.5 rounded text-[11px] font-semibold ${
                isTestPass ? 'bg-emerald-100 text-emerald-800' : 'bg-red-100 text-red-800'
              }`}
            >
              Tests: {submission.test_result}
            </span>
          </div>
        </div>
      </div>

      {/* Tabs Navigation */}
      <div className="border-b border-zinc-200">
        <nav className="flex space-x-6" role="tablist">
          <button
            type="button"
            role="tab"
            aria-selected={activeTab === 'preview'}
            onClick={() => setActiveTab('preview')}
            className={`pb-3 text-xs font-bold tracking-tight border-b-2 transition-colors ${
              activeTab === 'preview'
                ? 'border-blue-600 text-blue-600'
                : 'border-transparent text-zinc-500 hover:text-zinc-800 hover:border-zinc-300'
            }`}
          >
            Preview
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={activeTab === 'before_after'}
            onClick={() => setActiveTab('before_after')}
            className={`pb-3 text-xs font-bold tracking-tight border-b-2 transition-colors ${
              activeTab === 'before_after'
                ? 'border-blue-600 text-blue-600'
                : 'border-transparent text-zinc-500 hover:text-zinc-800 hover:border-zinc-300'
            }`}
          >
            Before/After
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={activeTab === 'evidence'}
            onClick={() => setActiveTab('evidence')}
            className={`pb-3 text-xs font-bold tracking-tight border-b-2 transition-colors ${
              activeTab === 'evidence'
                ? 'border-blue-600 text-blue-600'
                : 'border-transparent text-zinc-500 hover:text-zinc-800 hover:border-zinc-300'
            }`}
          >
            Evidence
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={activeTab === 'portfolio_card'}
            onClick={() => setActiveTab('portfolio_card')}
            className={`pb-3 text-xs font-bold tracking-tight border-b-2 transition-colors ${
              activeTab === 'portfolio_card'
                ? 'border-blue-600 text-blue-600'
                : 'border-transparent text-zinc-500 hover:text-zinc-800 hover:border-zinc-300'
            }`}
          >
            Portfolio Card
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={activeTab === 'lab_page'}
            onClick={() => setActiveTab('lab_page')}
            className={`pb-3 text-xs font-bold tracking-tight border-b-2 transition-colors ${
              activeTab === 'lab_page'
                ? 'border-blue-600 text-blue-600'
                : 'border-transparent text-zinc-500 hover:text-zinc-800 hover:border-zinc-300'
            }`}
          >
            Lab Page
          </button>
        </nav>
      </div>

      {/* Tab Contents */}
      <div className="mt-4">
        {/* 1. Preview Tab */}
        {activeTab === 'preview' && (
          <div role="tabpanel" data-testid="panel-preview" className="space-y-4">
            <PreviewFrame
              previewUrl={submission.preview_url}
              title={`${project.title} Preview`}
            />

            {sourceUrls.length > 0 && (
              <div
                data-testid="source-urls-block"
                className="bg-white rounded-2xl border border-zinc-200 p-6 shadow-sm space-y-3"
              >
                <div>
                  <h3 className="text-base font-bold text-zinc-900">Source-site Reference Links</h3>
                  <p className="text-xs text-zinc-500 mt-0.5">
                    Outbound references to the original third-party source site. Metadata only.
                  </p>
                </div>

                <ul className="space-y-1.5">
                  {sourceUrls.map((url) => (
                    <li key={url}>
                      <a
                        href={url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center space-x-1 text-xs text-blue-600 hover:text-blue-800 font-medium underline break-all"
                      >
                        <span>{url}</span>
                        <ExternalLink className="w-3.5 h-3.5 shrink-0" />
                      </a>
                    </li>
                  ))}
                </ul>

                <p
                  data-testid="origin-verdict"
                  className="text-xs text-zinc-600 bg-amber-50/50 p-3 rounded-lg border border-amber-200 leading-relaxed"
                >
                  Origin verdict: the linked source sites are original third-party products owned by
                  their respective companies. The reviewed artifact is an independent redesign
                  concept and is not affiliated with, sponsored by, or endorsed by the original
                  company.
                </p>
              </div>
            )}
          </div>
        )}

        {/* 2. Before / After Tab */}
        {activeTab === 'before_after' && (
          <div role="tabpanel" data-testid="panel-before_after" className="bg-white rounded-2xl border border-zinc-200 p-6 shadow-sm space-y-6">
            <div>
              <h3 className="text-base font-bold text-zinc-900">Before &amp; After Visual Audit</h3>
              <p className="text-xs text-zinc-500 mt-0.5">
                Comparison of original production interface and proposed redesign.
              </p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {/* Before */}
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-bold uppercase tracking-wider text-zinc-500">Before Redesign</span>
                  <span className="text-xs text-zinc-400 font-mono">{project.original_product}</span>
                </div>
                {meta.before_image_url ? (
                  <div className="border border-zinc-200 rounded-xl overflow-hidden bg-zinc-50 p-2">
                    <img
                      src={meta.before_image_url}
                      alt="Before redesign"
                      className="w-full h-auto rounded-lg object-contain max-h-96"
                    />
                  </div>
                ) : (
                  <div className="h-64 border-2 border-dashed border-zinc-200 rounded-xl flex items-center justify-center text-xs text-zinc-400">
                    No before screenshot provided.
                  </div>
                )}
              </div>

              {/* After */}
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-bold uppercase tracking-wider text-emerald-700">After Redesign</span>
                  <span className="text-xs text-zinc-400 font-mono">Revision {submission.revision}</span>
                </div>
                {meta.after_image_url ? (
                  <div className="border border-emerald-200 rounded-xl overflow-hidden bg-emerald-50/30 p-2">
                    <img
                      src={meta.after_image_url}
                      alt="After redesign"
                      className="w-full h-auto rounded-lg object-contain max-h-96"
                    />
                  </div>
                ) : (
                  <div className="h-64 border-2 border-dashed border-zinc-200 rounded-xl flex items-center justify-center text-xs text-zinc-400">
                    No after screenshot provided.
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* 3. Evidence Tab */}
        {activeTab === 'evidence' && (
          <div role="tabpanel" data-testid="panel-evidence">
            <EvidencePanel
              submission={submission}
              artifactSha256={artifact.sha256}
            />
          </div>
        )}

        {/* 4. Portfolio Card Tab */}
        {activeTab === 'portfolio_card' && (
          <div role="tabpanel" data-testid="panel-portfolio_card" className="bg-white rounded-2xl border border-zinc-200 p-6 shadow-sm space-y-6">
            <div>
              <h3 className="text-base font-bold text-zinc-900">Portfolio Card Representation</h3>
              <p className="text-xs text-zinc-500 mt-0.5">
                Preview of how this project will be listed in the Design Lab tab on the public portfolio.
              </p>
            </div>

            <div className="max-w-md mx-auto bg-white rounded-2xl border border-zinc-200 p-5 shadow-lg space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-xs font-semibold text-blue-600 bg-blue-50 px-2.5 py-0.5 rounded-full">
                  Design Lab
                </span>
                {meta.featured && (
                  <span className="text-xs font-semibold text-amber-700 bg-amber-50 px-2 py-0.5 rounded-full flex items-center gap-1">
                    <Sparkles className="w-3 h-3" /> Featured
                  </span>
                )}
              </div>

              <div>
                <h4 className="text-lg font-bold text-zinc-900">{project.title}</h4>
                <p className="text-xs text-zinc-600 mt-1">{meta.summary || project.original_product}</p>
                {meta.description && (
                  <p className="text-xs text-zinc-500 mt-2 line-clamp-2">{meta.description}</p>
                )}
              </div>

              {meta.metrics && Object.keys(meta.metrics).length > 0 && (
                <div className="grid grid-cols-2 gap-2 bg-zinc-50 p-3 rounded-xl border border-zinc-100">
                  {Object.entries(meta.metrics).map(([key, val]) => (
                    <div key={key} className="text-center">
                      <div className="text-sm font-bold text-zinc-900">{val}</div>
                      <div className="text-[10px] text-zinc-500 capitalize">{key.replace(/_/g, ' ')}</div>
                    </div>
                  ))}
                </div>
              )}

              {meta.tags && meta.tags.length > 0 && (
                <div className="flex flex-wrap gap-1.5 pt-1">
                  {meta.tags.map((tag) => (
                    <span
                      key={tag}
                      className="text-[11px] font-medium bg-zinc-100 text-zinc-700 px-2 py-0.5 rounded"
                    >
                      {tag}
                    </span>
                  ))}
                </div>
              )}

              <div className="pt-2 border-t border-zinc-100 flex items-center justify-between text-xs text-zinc-500">
                <span className="font-mono text-[11px]">Slug: {project.slug}</span>
                {meta.case_study_slug && (
                  <span className="text-blue-600 font-medium">Case study attached</span>
                )}
              </div>
            </div>
          </div>
        )}

        {/* 5. Lab Page Tab */}
        {activeTab === 'lab_page' && (
          <div role="tabpanel" data-testid="panel-lab_page" className="bg-white rounded-2xl border border-zinc-200 p-6 shadow-sm space-y-6">
            <div>
              <h3 className="text-base font-bold text-zinc-900">Lab Page Information</h3>
              <p className="text-xs text-zinc-500 mt-0.5">
                Public details rendered on the dedicated Lab page.
              </p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-4">
                <div>
                  <label className="text-xs font-bold uppercase tracking-wider text-zinc-500 block mb-1">
                    Lab Route Slug
                  </label>
                  <div className="font-mono text-sm bg-zinc-50 p-2.5 rounded-lg border border-zinc-200 text-zinc-800">
                    /lab/{project.slug}
                  </div>
                </div>

                <div>
                  <label className="text-xs font-bold uppercase tracking-wider text-zinc-500 block mb-1">
                    Original Product
                  </label>
                  <div className="text-sm font-medium text-zinc-800 bg-zinc-50 p-2.5 rounded-lg border border-zinc-200">
                    {project.original_product}
                  </div>
                </div>

                <div>
                  <label className="text-xs font-bold uppercase tracking-wider text-zinc-500 block mb-1">
                    Independent Disclaimer
                  </label>
                  <div className="text-xs text-zinc-600 bg-amber-50/50 p-3 rounded-lg border border-amber-200 leading-relaxed">
                    {project.disclaimer || 'Independent redesign exploration not affiliated with original creators.'}
                  </div>
                </div>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="text-xs font-bold uppercase tracking-wider text-zinc-500 block mb-1">
                    Focus Areas
                  </label>
                  <div className="flex flex-wrap gap-1.5">
                    {project.focus && project.focus.length > 0 ? (
                      project.focus.map((item) => (
                        <span key={item} className="text-xs bg-zinc-100 text-zinc-700 px-2.5 py-1 rounded-md border border-zinc-200">
                          {item}
                        </span>
                      ))
                    ) : (
                      <span className="text-xs text-zinc-400">No focus tags specified</span>
                    )}
                  </div>
                </div>

                <div>
                  <label className="text-xs font-bold uppercase tracking-wider text-zinc-500 block mb-1">
                    Supported Platforms
                  </label>
                  <div className="flex flex-wrap gap-1.5">
                    {project.platforms && project.platforms.length > 0 ? (
                      project.platforms.map((plat) => (
                        <span key={plat} className="text-xs bg-zinc-100 text-zinc-700 px-2.5 py-1 rounded-md border border-zinc-200">
                          {plat}
                        </span>
                      ))
                    ) : (
                      <span className="text-xs text-zinc-400">Web / Mobile</span>
                    )}
                  </div>
                </div>

                <div>
                  <label className="text-xs font-bold uppercase tracking-wider text-zinc-500 block mb-1">
                    Live Sandbox Artifact Preview
                  </label>
                  <a
                    href={submission.preview_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center space-x-1 text-xs text-blue-600 hover:text-blue-800 font-medium underline"
                  >
                    <span>{submission.preview_url}</span>
                    <ExternalLink className="w-3.5 h-3.5" />
                  </a>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Approval Step-up Dialog Modal */}
      {isApprovalOpen && (
        <ApprovalDialog
          onClose={() => setIsApprovalOpen(false)}
          onConfirm={handleApproveConfirm}
          artifactSha256={artifact.sha256}
          projectTitle={project.title}
          revision={submission.revision}
          isSubmitting={isProcessing}
        />
      )}
    </div>
  );
}
