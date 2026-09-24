import { useState } from 'react';
import { Smartphone, Monitor, ExternalLink, ShieldCheck } from 'lucide-react';

interface PreviewFrameProps {
  previewUrl: string;
  title?: string;
}

export function PreviewFrame({ previewUrl, title = 'Project Preview' }: PreviewFrameProps) {
  const [viewport, setViewport] = useState<'desktop' | 'mobile'>('desktop');

  return (
    <div className="flex flex-col h-full bg-zinc-100 rounded-xl border border-zinc-200 overflow-hidden shadow-sm">
      {/* Viewport & Security Toolbar */}
      <div className="flex flex-wrap items-center justify-between gap-3 px-4 py-2.5 bg-white border-b border-zinc-200">
        <div className="flex items-center space-x-2">
          <div className="flex items-center bg-zinc-100 p-1 rounded-lg border border-zinc-200">
            <button
              type="button"
              onClick={() => setViewport('desktop')}
              aria-label="Desktop viewport"
              className={`flex items-center space-x-1.5 px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                viewport === 'desktop'
                  ? 'bg-white text-zinc-900 shadow-sm'
                  : 'text-zinc-600 hover:text-zinc-900'
              }`}
            >
              <Monitor className="w-3.5 h-3.5" />
              <span>Desktop</span>
            </button>
            <button
              type="button"
              onClick={() => setViewport('mobile')}
              aria-label="Mobile viewport"
              className={`flex items-center space-x-1.5 px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                viewport === 'mobile'
                  ? 'bg-white text-zinc-900 shadow-sm'
                  : 'text-zinc-600 hover:text-zinc-900'
              }`}
            >
              <Smartphone className="w-3.5 h-3.5" />
              <span>Mobile (375px)</span>
            </button>
          </div>
        </div>

        {/* Security sandbox disclaimer & External link */}
        <div className="flex items-center space-x-3 text-xs">
          <span className="flex items-center space-x-1 text-emerald-700 bg-emerald-50 px-2 py-1 rounded border border-emerald-200 font-mono text-[11px]">
            <ShieldCheck className="w-3.5 h-3.5 mr-0.5 text-emerald-600" />
            sandbox="allow-scripts allow-forms" (no same-origin)
          </span>
          {previewUrl && (
            <a
              href={previewUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center space-x-1 text-zinc-600 hover:text-zinc-900 transition-colors"
            >
              <span>Open raw</span>
              <ExternalLink className="w-3.5 h-3.5" />
            </a>
          )}
        </div>
      </div>

      {/* Frame Container */}
      <div className="flex-grow p-4 flex items-center justify-center min-h-[620px] bg-zinc-50 overflow-auto">
        <div
          data-testid="preview-container"
          className={`transition-all duration-200 ease-in-out bg-white shadow-md border border-zinc-200 flex flex-col ${
            viewport === 'desktop'
              ? 'preview-container-desktop desktop w-full h-[640px] rounded-lg'
              : 'preview-container-mobile mobile w-[375px] h-[667px] rounded-[36px] p-3 border-4 border-zinc-800 shadow-xl'
          }`}
        >
          {/* Mobile phone mock top notch if mobile */}
          {viewport === 'mobile' && (
            <div className="w-full flex justify-center py-1">
              <div className="w-24 h-4 bg-zinc-800 rounded-full mb-1"></div>
            </div>
          )}

          <iframe
            data-testid="preview-iframe"
            title={title}
            src={previewUrl}
            sandbox="allow-scripts allow-forms"
            className={`w-full h-full border-0 ${viewport === 'mobile' ? 'rounded-[24px]' : 'rounded-b-lg'}`}
          />
        </div>
      </div>
    </div>
  );
}
