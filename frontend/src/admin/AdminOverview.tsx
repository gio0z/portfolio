import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { adminApi, AdminApiError } from './AdminApi';
import type { OverviewResponse } from './types';

export function AdminOverview() {
  const [overview, setOverview] = useState<OverviewResponse | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const handleRefresh = async () => {
    setLoading(true);
    try {
      const data = await adminApi.getOverview();
      setOverview(data);
      setError(null);
    } catch (err) {
      if (err instanceof AdminApiError) {
        setError(err.message);
      } else {
        setError('Failed to load admin overview metrics.');
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    let active = true;
    adminApi
      .getOverview()
      .then((data) => {
        if (active) {
          setOverview(data);
          setError(null);
        }
      })
      .catch((err) => {
        if (active) {
          if (err instanceof AdminApiError) {
            setError(err.message);
          } else {
            setError('Failed to load admin overview metrics.');
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
  }, []);

  if (loading) {
    return (
      <div className="p-8 text-center text-zinc-600">
        <p>Loading overview metrics...</p>
      </div>
    );
  }

  if (error || !overview) {
    return (
      <div className="space-y-4">
        <div className="p-4 bg-red-50 border border-red-200 text-red-700 rounded-lg text-sm" role="alert">
          {error || 'Unable to load dashboard data.'}
        </div>
        <button
          onClick={handleRefresh}
          className="px-4 py-2 text-sm bg-zinc-800 text-white rounded-md hover:bg-zinc-700 font-medium"
        >
          Try Again
        </button>
      </div>
    );
  }

  const latestDeployment = overview.recent_audits.find(
    (a) =>
      a.action?.toLowerCase().includes('publish') ||
      a.action?.toLowerCase().includes('approve') ||
      a.action?.toLowerCase().includes('deploy')
  );

  return (
    <div className="space-y-8">
      {/* Top Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-zinc-900">Admin Dashboard</h2>
          <p className="text-sm text-zinc-500">
            System overview of Design Lab submissions, publication status, and audit logs.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Link
            to="/admin/reviews"
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg shadow-sm transition-colors"
          >
            Go to Review Queue ({overview.pending_reviews})
          </Link>
          <button
            onClick={handleRefresh}
            className="px-3 py-2 text-sm bg-zinc-100 hover:bg-zinc-200 text-zinc-700 rounded-lg font-medium transition-colors"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Metrics Cards Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        {/* Card 1: Pending Reviews */}
        <div className="bg-white p-5 rounded-xl border border-zinc-200 shadow-sm flex flex-col justify-between">
          <div>
            <span className="text-xs font-semibold tracking-wider text-zinc-400 uppercase">
              Pending Reviews
            </span>
            <div className="text-3xl font-bold text-zinc-900 mt-2">
              {overview.pending_reviews}
            </div>
          </div>
          <div className="mt-4 pt-3 border-t border-zinc-100 text-xs text-zinc-500 flex items-center justify-between">
            <span>Awaiting owner action</span>
            <Link to="/admin/reviews" className="text-blue-600 hover:underline font-medium">
              View &rarr;
            </Link>
          </div>
        </div>

        {/* Card 2: Published Projects */}
        <div className="bg-white p-5 rounded-xl border border-zinc-200 shadow-sm flex flex-col justify-between">
          <div>
            <span className="text-xs font-semibold tracking-wider text-zinc-400 uppercase">
              Published Projects
            </span>
            <div className="text-3xl font-bold text-emerald-600 mt-2">
              {overview.published_projects}
            </div>
          </div>
          <div className="mt-4 pt-3 border-t border-zinc-100 text-xs text-zinc-500">
            Live on Lab & Portfolio
          </div>
        </div>

        {/* Card 3: Build Failures */}
        <div className="bg-white p-5 rounded-xl border border-zinc-200 shadow-sm flex flex-col justify-between">
          <div>
            <span className="text-xs font-semibold tracking-wider text-zinc-400 uppercase">
              Build Failures
            </span>
            <div
              className={`text-3xl font-bold mt-2 ${
                overview.build_failures > 0 ? 'text-red-600' : 'text-zinc-900'
              }`}
            >
              {overview.build_failures}
            </div>
          </div>
          <div className="mt-4 pt-3 border-t border-zinc-100 text-xs text-zinc-500">
            Verification or scan failures
          </div>
        </div>

        {/* Card 4: Latest Deployment */}
        <div className="bg-white p-5 rounded-xl border border-zinc-200 shadow-sm flex flex-col justify-between">
          <div>
            <span className="text-xs font-semibold tracking-wider text-zinc-400 uppercase">
              Latest Deployment
            </span>
            <div className="text-sm font-semibold text-zinc-800 mt-2 truncate">
              {latestDeployment ? (
                <span title={`${latestDeployment.action} (${latestDeployment.result})`}>
                  {latestDeployment.action}
                </span>
              ) : (
                <span className="text-zinc-400">None yet</span>
              )}
            </div>
          </div>
          <div className="mt-4 pt-3 border-t border-zinc-100 text-xs text-zinc-500 truncate">
            {latestDeployment?.timestamp
              ? new Date(latestDeployment.timestamp).toLocaleString()
              : 'No deployments recorded'}
          </div>
        </div>
      </div>

      {/* Recent MCP & Audit Activity */}
      <div className="bg-white border border-zinc-200 rounded-xl shadow-sm overflow-hidden">
        <div className="px-6 py-4 border-b border-zinc-200 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-zinc-900">Recent Audit & MCP Activity</h3>
          <span className="text-xs text-zinc-500">Last {overview.recent_audits.length} events</span>
        </div>

        {overview.recent_audits.length === 0 ? (
          <div className="p-8 text-center text-zinc-500 text-sm">
            No recent activity recorded.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-zinc-200 text-left text-sm">
              <thead className="bg-zinc-50 text-xs font-semibold text-zinc-500 uppercase tracking-wider">
                <tr>
                  <th scope="col" className="px-6 py-3">Timestamp</th>
                  <th scope="col" className="px-6 py-3">Actor</th>
                  <th scope="col" className="px-6 py-3">Action</th>
                  <th scope="col" className="px-6 py-3">Target</th>
                  <th scope="col" className="px-6 py-3">Result</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-100 bg-white">
                {overview.recent_audits.map((event, idx) => (
                  <tr key={`${event.request_id}-${idx}`} className="hover:bg-zinc-50 transition-colors">
                    <td className="px-6 py-3.5 whitespace-nowrap text-xs text-zinc-500">
                      {new Date(event.timestamp).toLocaleString()}
                    </td>
                    <td className="px-6 py-3.5 whitespace-nowrap">
                      <div className="flex items-center gap-1.5">
                        <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-zinc-100 text-zinc-600">
                          {event.actor_kind}
                        </span>
                        <span className="font-medium text-zinc-800">{event.actor_id}</span>
                      </div>
                    </td>
                    <td className="px-6 py-3.5 whitespace-nowrap text-zinc-700 font-mono text-xs">
                      {event.action}
                    </td>
                    <td className="px-6 py-3.5 whitespace-nowrap text-xs text-zinc-500 font-mono">
                      {event.submission_id || event.project_id || '-'}
                    </td>
                    <td className="px-6 py-3.5 whitespace-nowrap">
                      <span
                        className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold ${
                          event.result?.toLowerCase() === 'success' ||
                          event.result?.toLowerCase() === 'passed'
                            ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
                            : 'bg-red-50 text-red-700 border border-red-200'
                        }`}
                      >
                        {event.result}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
