import { useState } from 'react';
import { ShieldAlert, Globe, X, Lock } from 'lucide-react';

interface ApprovalDialogProps {
  onClose: () => void;
  onConfirm: () => Promise<void>;
  artifactSha256: string;
  projectTitle: string;
  revision: number;
  isSubmitting?: boolean;
}

export function ApprovalDialog({
  onClose,
  onConfirm,
  artifactSha256,
  projectTitle,
  revision,
  isSubmitting = false,
}: ApprovalDialogProps) {
  const [stepUpText, setStepUpText] = useState('');

  const isConfirmed = stepUpText.trim() === 'CONFIRM';

  const handleSubmit = async (e: React.SubmitEvent) => {
    e.preventDefault();
    if (!isConfirmed || isSubmitting) return;
    await onConfirm();
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="approval-dialog-title"
      className="fixed inset-0 z-50 overflow-y-auto bg-black/60 backdrop-blur-sm flex items-center justify-center p-4"
    >
      <div className="bg-white rounded-2xl shadow-2xl border border-zinc-200 w-full max-w-xl overflow-hidden transition-all transform animate-in fade-in zoom-in-95">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-zinc-200 bg-zinc-50">
          <div className="flex items-center space-x-2.5">
            <div className="p-2 bg-emerald-100 rounded-lg text-emerald-700">
              <Lock className="w-5 h-5" />
            </div>
            <div>
              <h3 id="approval-dialog-title" className="text-base font-bold text-zinc-900">
                Approve &amp; Publish Project
              </h3>
              <p className="text-xs text-zinc-500">Owner step-up authorization required</p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={isSubmitting}
            className="text-zinc-400 hover:text-zinc-600 p-1.5 rounded-lg hover:bg-zinc-100 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <form onSubmit={handleSubmit} className="p-6 space-y-5">
          {/* Dual Destination Notice */}
          <div className="bg-blue-50 border border-blue-200 rounded-xl p-4">
            <h4 className="text-xs font-bold text-blue-900 uppercase tracking-wider mb-2 flex items-center gap-1.5">
              <Globe className="w-4 h-4 text-blue-700" />
              Atomic Dual-Destination Publication
            </h4>
            <p className="text-xs text-blue-800 leading-relaxed">
              This action atomically deploys and activates <strong>"{projectTitle}"</strong> (Revision {revision}) to both destinations simultaneously:
            </p>
            <ul
              data-testid="publication-destinations"
              className="mt-2.5 space-y-1.5 text-xs text-blue-900 font-medium"
            >
              <li className="flex items-center gap-2">
                <span className="w-1.5 h-1.5 rounded-full bg-blue-600"></span>
                <strong>Design Lab</strong> interactive sandbox catalog (live executable artifact)
              </li>
              <li className="flex items-center gap-2">
                <span className="w-1.5 h-1.5 rounded-full bg-blue-600"></span>
                <strong>Portfolio</strong> Design Lab tab showcasing the verified engineering case study
              </li>
            </ul>
          </div>

          {/* Bound Artifact Hash */}
          <div className="bg-zinc-900 text-zinc-100 p-4 rounded-xl border border-zinc-800">
            <span className="text-[11px] uppercase tracking-wider text-zinc-400 font-semibold block mb-1">
              Exact Reviewed Artifact Hash
            </span>
            <code className="text-xs sm:text-sm font-mono text-emerald-400 break-all select-all font-semibold block">
              {artifactSha256}
            </code>
            <p className="text-[11px] text-zinc-400 mt-2">
              Publication will strictly fail if the deployment target checksum does not match this exact hash.
            </p>
          </div>

          {/* Step-up verification */}
          <div className="space-y-2">
            <label htmlFor="step-up-input" className="block text-xs font-semibold text-zinc-700 flex items-center justify-between">
              <span>Step-Up Verification</span>
              <span className="text-zinc-400 font-normal">Type <strong>CONFIRM</strong> to proceed</span>
            </label>
            <input
              id="step-up-input"
              data-testid="step-up-input"
              type="text"
              value={stepUpText}
              onChange={(e) => setStepUpText(e.target.value)}
              placeholder="CONFIRM"
              disabled={isSubmitting}
              autoComplete="off"
              className="w-full px-3.5 py-2 border border-zinc-300 rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 bg-white"
            />
          </div>

          {/* Safety warning */}
          <div className="flex items-start space-x-2 text-[11px] text-zinc-500">
            <ShieldAlert className="w-4 h-4 text-amber-500 shrink-0 mt-0.5" />
            <span>
              This operation executes owner step-up authorization, updates the public registry, and records an immutable audit log entry.
            </span>
          </div>

          {/* Actions */}
          <div className="flex items-center justify-end space-x-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              disabled={isSubmitting}
              className="px-4 py-2 text-xs font-semibold text-zinc-700 bg-white border border-zinc-300 rounded-lg hover:bg-zinc-50 transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!isConfirmed || isSubmitting}
              className={`px-4 py-2 text-xs font-semibold rounded-lg shadow-sm transition-all ${
                isConfirmed && !isSubmitting
                  ? 'bg-emerald-600 text-white hover:bg-emerald-700'
                  : 'bg-zinc-200 text-zinc-400 cursor-not-allowed'
              }`}
            >
              {isSubmitting ? 'Publishing...' : 'Confirm & Publish'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
