import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { adminApi, AdminApiError } from './AdminApi';
import type { ReviewQueueItem } from './types';

interface ReviewQueueProps {
  initialSubmissions?: ReviewQueueItem[];
}

export function ReviewQueue({ initialSubmissions }: ReviewQueueProps) {
  const [submissions, setSubmissions] = useState<ReviewQueueItem[]>(initialSubmissions || []);
  const [loading, setLoading] = useState<boolean>(!initialSubmissions);
  const [error, setError] = useState<string | null>(null);
  const [actionSuccess, setActionSuccess] = useState<string | null>(null);
  const [activeActionId, setActiveActionId] = useState<string | null>(null);

  const handleRefresh = async () => {
    setLoading(true);
    try {
      const data = await adminApi.getReviews();
      setSubmissions(data || []);
      setError(null);
    } catch (err) {
      if (err instanceof AdminApiError) {
        setError(err.message);
      } else {
        setError('Failed to load review queue submissions.');
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!initialSubmissions) {
      let active = true;
      adminApi
        .getReviews()
        .then((data) => {
          if (active) {
            setSubmissions(data || []);
            setError(null);
          }
        })
        .catch((err) => {
          if (active) {
            if (err instanceof AdminApiError) {
              setError(err.message);
            } else {
              setError('Failed to load review queue submissions.');
            }
          }
        })
        .finally(() => {
          if (active) {
            setLoading(false);
          }
        });

      return () => {
        active = false;
      };
    }
  }, [initialSubmissions]);

  const handleApprove = async (item: ReviewQueueItem) => {
    setActiveActionId(item.id);
    setError(null);
    setActionSuccess(null);
    try {
      const res = await adminApi.approveAndPublish(item.id, item.artifact_sha256);
      setActionSuccess(`Successfully approved and published "${item.project_title}" (Revision ${res.revision}).`);
      await handleRefresh();
    } catch (err) {
      if (err instanceof AdminApiError) {
        setError(err.message);
      } else {
        setError('Failed to approve and publish submission.');
      }
    } finally {
      setActiveActionId(null);
    }
  };

  const handleRequestChanges = async (item: ReviewQueueItem) => {
    const reason = window.prompt('Please enter feedback for requesting changes:');
    if (!reason || !reason.trim()) return;

    setActiveActionId(item.id);
    setError(null);
    setActionSuccess(null);
    try {
      await adminApi.requestChanges(item.id, reason.trim());
      setActionSuccess(`Changes requested for "${item.project_title}".`);
      await handleRefresh();
    } catch (err) {
      if (err instanceof AdminApiError) {
        setError(err.message);
      } else {
        setError('Failed to request changes.');
      }
    } finally {
      setActiveActionId(null);
    }
  };

  const handleReject = async (item: ReviewQueueItem) => {
    const reason = window.prompt('Please enter reason for rejecting this submission:');
    if (!reason || !reason.trim()) return;

    setActiveActionId(item.id);
    setError(null);
    setActionSuccess(null);
    try {
      await adminApi.reject(item.id, reason.trim());
      setActionSuccess(`Rejected "${item.project_title}".`);
      await handleRefresh();
    } catch (err) {
      if (err instanceof AdminApiError) {
        setError(err.message);
      } else {
        setError('Failed to reject submission.');
      }
    } finally {
      setActiveActionId(null);
    }
  };

  if (loading) {
    return (
      <div className="p-8 text-center text-zinc-600">
        <p>Loading review queue...</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold text-zinc-900">Review Queue</h2>
          <p className="text-sm text-zinc-500">
            Audit and approve pending lab artifacts before production publication.
          </p>
        </div>
        <button
          onClick={handleRefresh}
          className="px-3 py-1.5 text-sm bg-zinc-100 hover:bg-zinc-200 text-zinc-700 rounded-md font-medium transition-colors"
        >
          Refresh
        </button>
      </div>

      {error && (
        <div className="p-4 bg-red-50 border border-red-200 text-red-700 rounded-lg text-sm" role="alert">
          {error}
        </div>
      )}

      {actionSuccess && (
        <div className="p-4 bg-emerald-50 border border-emerald-200 text-emerald-700 rounded-lg text-sm" role="status">
          {actionSuccess}
        </div>
      )}

      {submissions.length === 0 ? (
        <div className="p-12 text-center border-2 border-dashed border-zinc-200 rounded-xl">
          <p className="text-zinc-500 font-medium">No submissions in review queue</p>
          <p className="text-xs text-zinc-400 mt-1">All proposed designs and revisions have been processed.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {submissions.map((item) => {
            const isApproveEligible =
              item.can_approve &&
              item.security_scan_result?.toLowerCase() === 'passed' &&
              item.build_result?.toLowerCase() === 'passed' &&
              item.test_result?.toLowerCase() === 'passed';

            const isProcessing = activeActionId === item.id;

            return (
              <div
                key={item.id}
                data-testid={`review-item-${item.id}`}
                className="bg-white border border-zinc-200 rounded-xl p-6 shadow-sm hover:border-zinc-300 transition-shadow space-y-4"
              >
                <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="text-lg font-semibold text-zinc-900">{item.project_title}</h3>
                      <span className="text-xs px-2 py-0.5 rounded bg-zinc-100 text-zinc-600 font-mono">
                        rev {item.revision}
                      </span>
                    </div>
                    <p className="text-sm text-zinc-500 mt-0.5">
                      Original Product: <span className="font-medium text-zinc-700">{item.original_product}</span>
                    </p>
                  </div>

                  <div className="flex items-center gap-2">
                    <span
                      className={`inline-flex items-center px-2.5 py-1 rounded-full text-xs font-semibold ${
                        item.state === 'IN_REVIEW'
                          ? 'bg-amber-100 text-amber-800'
                          : item.state === 'APPROVED' || item.state === 'PUBLISHED'
                          ? 'bg-emerald-100 text-emerald-800'
                          : item.state === 'REJECTED'
                          ? 'bg-red-100 text-red-800'
                          : 'bg-zinc-100 text-zinc-800'
                      }`}
                    >
                      {item.state}
                    </span>
                  </div>
                </div>

                {/* Evidence & Verification Badges */}
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 pt-2 border-t border-zinc-100">
                  <div className="flex items-center gap-2 text-xs">
                    <span className="text-zinc-500">Build:</span>
                    <span
                      className={`font-semibold px-2 py-0.5 rounded ${
                        item.build_result?.toLowerCase() === 'passed'
                          ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
                          : 'bg-red-50 text-red-700 border border-red-200'
                      }`}
                    >
                      Build: {item.build_result || 'unknown'}
                    </span>
                  </div>

                  <div className="flex items-center gap-2 text-xs">
                    <span className="text-zinc-500">Test:</span>
                    <span
                      className={`font-semibold px-2 py-0.5 rounded ${
                        item.test_result?.toLowerCase() === 'passed'
                          ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
                          : 'bg-red-50 text-red-700 border border-red-200'
                      }`}
                    >
                      Test: {item.test_result || 'unknown'}
                    </span>
                  </div>

                  <div className="flex items-center gap-2 text-xs">
                    <span className="text-zinc-500">Security:</span>
                    <span
                      className={`font-semibold px-2 py-0.5 rounded ${
                        item.security_scan_result?.toLowerCase() === 'passed'
                          ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
                          : 'bg-red-50 text-red-700 border border-red-200'
                      }`}
                    >
                      Scan: {item.security_scan_result || 'unknown'}
                    </span>
                  </div>
                </div>

                {/* Artifact SHA */}
                <div className="text-xs text-zinc-500 font-mono bg-zinc-50 p-2 rounded border border-zinc-100 flex items-center justify-between overflow-x-auto">
                  <span>
                    SHA256: <span className="text-zinc-800">{item.artifact_sha256}</span>
                  </span>
                  {item.preview_url && (
                    <a
                      href={item.preview_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="ml-4 text-blue-600 hover:text-blue-800 font-sans font-medium underline"
                    >
                      Preview
                    </a>
                  )}
                </div>

                {/* Actions */}
                <div className="flex flex-wrap items-center justify-end gap-3 pt-2">
                  <Link
                    to={`/admin/reviews/${item.id}`}
                    aria-label={`Review ${item.project_title}`}
                    className="px-3 py-1.5 text-xs font-medium text-zinc-700 bg-white border border-zinc-300 rounded-md hover:bg-zinc-50 transition-colors"
                  >
                    Review
                  </Link>

                  <button
                    type="button"
                    onClick={() => handleRequestChanges(item)}
                    disabled={isProcessing}
                    className="px-3 py-1.5 text-xs font-medium text-amber-700 bg-amber-50 border border-amber-300 rounded-md hover:bg-amber-100 transition-colors disabled:opacity-50"
                  >
                    Request Changes
                  </button>

                  <button
                    type="button"
                    onClick={() => handleReject(item)}
                    disabled={isProcessing}
                    className="px-3 py-1.5 text-xs font-medium text-red-700 bg-red-50 border border-red-300 rounded-md hover:bg-red-100 transition-colors disabled:opacity-50"
                  >
                    Reject
                  </button>

                  <button
                    type="button"
                    onClick={() => handleApprove(item)}
                    disabled={!isApproveEligible || isProcessing}
                    title={
                      !isApproveEligible
                        ? 'All checks (build, test, and security scan) must pass before approval.'
                        : 'Approve and publish to production'
                    }
                    className={`px-3 py-1.5 text-xs font-semibold rounded-md transition-colors ${
                      isApproveEligible && !isProcessing
                        ? 'bg-emerald-600 text-white hover:bg-emerald-700 shadow-sm'
                        : 'bg-zinc-200 text-zinc-400 cursor-not-allowed'
                    }`}
                  >
                    {isProcessing ? 'Processing...' : 'Approve'}
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
