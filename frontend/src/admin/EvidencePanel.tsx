import { CheckCircle2, XCircle, AlertTriangle, Shield, FileCode, HardDrive } from 'lucide-react';
import type { SubmissionDetail } from './types';

interface EvidencePanelProps {
  submission: SubmissionDetail;
  artifactSha256: string;
}

export function EvidencePanel({ submission, artifactSha256 }: EvidencePanelProps) {
  const meta = submission.portfolio_metadata || {};
  const isBuildPass = submission.build_result?.toLowerCase() === 'passed';
  const isTestPass = submission.test_result?.toLowerCase() === 'passed';
  const isScanPass = submission.security_scan_result?.toLowerCase() === 'passed';

  return (
    <div className="space-y-6">
      {/* Prominent Artifact Hash */}
      <div className="bg-zinc-900 text-zinc-100 p-4 rounded-xl shadow-sm border border-zinc-800">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
          <div>
            <span className="text-xs uppercase tracking-wider text-zinc-400 font-semibold flex items-center gap-1.5">
              <HardDrive className="w-4 h-4 text-zinc-400" />
              Reviewed Immutable Artifact Hash (SHA-256)
            </span>
            <div className="font-mono text-sm sm:text-base text-emerald-400 break-all select-all font-semibold mt-1">
              {artifactSha256}
            </div>
          </div>
          <div className="self-start sm:self-center">
            <span className="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-zinc-800 text-zinc-300 border border-zinc-700">
              Revision {submission.revision}
            </span>
          </div>
        </div>
      </div>

      {/* CI & Security Status Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* Build Status Card */}
        <div className="bg-white p-4 rounded-xl border border-zinc-200 shadow-sm flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-zinc-500 uppercase tracking-wider">Build</span>
              {isBuildPass ? (
                <CheckCircle2 className="w-5 h-5 text-emerald-600" />
              ) : (
                <XCircle className="w-5 h-5 text-red-600" />
              )}
            </div>
            <div
              data-testid="evidence-build-status"
              className={`text-lg font-bold capitalize mt-2 ${
                isBuildPass ? 'text-emerald-700' : 'text-red-700'
              }`}
            >
              {submission.build_result || 'Unknown'}
            </div>
          </div>
          {meta.build_logs && (
            <div className="mt-3 bg-zinc-50 p-2.5 rounded-lg border border-zinc-200 text-xs font-mono text-zinc-700 whitespace-pre-wrap max-h-32 overflow-y-auto">
              {meta.build_logs}
            </div>
          )}
        </div>

        {/* Test Status Card */}
        <div className="bg-white p-4 rounded-xl border border-zinc-200 shadow-sm flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-zinc-500 uppercase tracking-wider">Automated Tests</span>
              {isTestPass ? (
                <CheckCircle2 className="w-5 h-5 text-emerald-600" />
              ) : (
                <XCircle className="w-5 h-5 text-red-600" />
              )}
            </div>
            <div
              data-testid="evidence-test-status"
              className={`text-lg font-bold capitalize mt-2 ${
                isTestPass ? 'text-emerald-700' : 'text-red-700'
              }`}
            >
              {submission.test_result || 'Unknown'}
            </div>
          </div>
          {meta.test_logs && (
            <div className="mt-3 bg-zinc-50 p-2.5 rounded-lg border border-zinc-200 text-xs font-mono text-zinc-700 whitespace-pre-wrap max-h-32 overflow-y-auto">
              {meta.test_logs}
            </div>
          )}
        </div>

        {/* Security Scan Card */}
        <div className="bg-white p-4 rounded-xl border border-zinc-200 shadow-sm flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-zinc-500 uppercase tracking-wider flex items-center gap-1">
                <Shield className="w-3.5 h-3.5 text-zinc-400" />
                Security Scan
              </span>
              {isScanPass ? (
                <CheckCircle2 className="w-5 h-5 text-emerald-600" />
              ) : (
                <AlertTriangle className="w-5 h-5 text-red-600" />
              )}
            </div>
            <div
              data-testid="evidence-scan-status"
              className={`text-lg font-bold capitalize mt-2 ${
                isScanPass ? 'text-emerald-700' : 'text-red-700'
              }`}
            >
              {submission.security_scan_result || 'Unknown'}
            </div>
          </div>
          {meta.security_scan_logs && (
            <div className="mt-3 bg-zinc-50 p-2.5 rounded-lg border border-zinc-200 text-xs font-mono text-zinc-700 whitespace-pre-wrap max-h-32 overflow-y-auto">
              {meta.security_scan_logs}
            </div>
          )}
        </div>
      </div>

      {/* Artifact Assets / Evidence List */}
      {meta.assets && meta.assets.length > 0 && (
        <div className="bg-white p-5 rounded-xl border border-zinc-200 shadow-sm">
          <h4 className="text-sm font-semibold text-zinc-900 mb-3 flex items-center gap-1.5">
            <FileCode className="w-4 h-4 text-zinc-500" />
            Verified Artifact Assets ({meta.assets.length})
          </h4>
          <div className="divide-y divide-zinc-100 border border-zinc-200 rounded-lg overflow-hidden">
            {meta.assets.map((asset, idx) => (
              <div key={idx} className="p-3 flex items-center justify-between text-xs hover:bg-zinc-50 transition-colors">
                <div className="flex items-center space-x-2">
                  <span className="font-mono text-zinc-800 font-medium">{asset.name}</span>
                  {asset.size && (
                    <span className="text-zinc-400">
                      ({Math.round(asset.size / 1024)} KB)
                    </span>
                  )}
                </div>
                {asset.url && (
                  <a
                    href={asset.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-600 hover:text-blue-800 underline font-medium"
                  >
                    View Asset
                  </a>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
