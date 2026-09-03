import React, { useState } from 'react';
import { ArrowDownRight, Mail, Phone, Send, MapPin, CheckCircle, AlertCircle, Loader2 } from 'lucide-react';
import type { ContactFormData, ContactResponse } from '../types';

export const ContactSection: React.FC = () => {
  const [formData, setFormData] = useState<ContactFormData>({
    name: '',
    email: '',
    subject: '',
    message: '',
  });

  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<ContactResponse | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setErrorMsg(null);
    setResult(null);

    try {
      const res = await fetch('/api/contact', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(formData),
      });

      const data = await res.json();

      if (!res.ok) {
        throw new Error(data.error || 'Failed to submit message');
      }

      setResult(data);
      setFormData({ name: '', email: '', subject: '', message: '' });
    } catch (err: unknown) {
      if (err instanceof Error) {
        setErrorMsg(err.message);
      } else {
        setErrorMsg('An unexpected error occurred while sending message');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <section id="contact" className="py-24 sm:py-32 px-4 sm:px-8 max-w-7xl mx-auto">
      <div className="bg-[#18181B] text-white rounded-[32px] p-8 sm:p-14 lg:p-18 shadow-2xl border border-white/5 relative overflow-hidden">
        {/* Glow Accent */}
        <div className="absolute top-0 right-0 w-96 h-96 bg-blue-600/10 blur-[120px] rounded-full pointer-events-none" />

        <div className="grid grid-cols-1 lg:grid-cols-12 gap-12 items-start relative z-10">
          {/* Left Column: Direct Info */}
          <div className="lg:col-span-5">
            <div className="inline-flex items-center gap-2 text-xs font-semibold tracking-wider uppercase text-zinc-400 mb-6 font-mono">
              <span className="flex items-center justify-center w-4 h-4 rounded bg-blue-600 text-white">
                <ArrowDownRight className="w-3 h-3" />
              </span>
              <span>Initiate Contact</span>
            </div>

            <h2 className="text-4xl sm:text-5xl font-extrabold tracking-tight leading-[1.1] mb-6">
              <span>Let's Build.</span> <br />
              <span className="text-zinc-500 font-bold">Something Great.</span>
            </h2>

            <p className="text-zinc-400 text-sm sm:text-base leading-relaxed mb-10 max-w-md">
              Whether you need a high-concurrency Go backend, ultra-fast reactive frontend, or autonomous AI agent mesh, reach out directly.
            </p>

            <div className="space-y-4">
              <a
                href="mailto:regio@zoo.com"
                className="flex items-center gap-4 p-4 rounded-2xl bg-[#242428] hover:bg-[#2A2A30] border border-white/5 hover:border-blue-500/40 transition-all group"
              >
                <div className="w-10 h-10 rounded-xl bg-blue-600/20 text-blue-400 flex items-center justify-center">
                  <Mail className="w-5 h-5" />
                </div>
                <div>
                  <div className="text-xs text-zinc-400 font-mono">EMAIL</div>
                  <div className="text-sm font-semibold text-white group-hover:text-blue-400 transition-colors">
                    regio@zoo.com
                  </div>
                </div>
              </a>

              <a
                href="https://wa.me/6285156439303"
                target="_blank"
                rel="noreferrer"
                className="flex items-center gap-4 p-4 rounded-2xl bg-[#242428] hover:bg-[#2A2A30] border border-white/5 hover:border-emerald-500/40 transition-all group"
              >
                <div className="w-10 h-10 rounded-xl bg-emerald-600/20 text-emerald-400 flex items-center justify-center">
                  <Phone className="w-5 h-5" />
                </div>
                <div>
                  <div className="text-xs text-zinc-400 font-mono">WHATSAPP</div>
                  <div className="text-sm font-semibold text-white group-hover:text-emerald-400 transition-colors">
                    +62 851-5643-9303
                  </div>
                </div>
              </a>

              <div className="flex items-center gap-4 p-4 rounded-2xl bg-[#242428] border border-white/5">
                <div className="w-10 h-10 rounded-xl bg-zinc-800 text-zinc-400 flex items-center justify-center">
                  <MapPin className="w-5 h-5" />
                </div>
                <div>
                  <div className="text-xs text-zinc-400 font-mono">LOCATION</div>
                  <div className="text-sm font-semibold text-white">
                    Indonesia (UTC+07:00 / WIB)
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Right Column: Interactive Form */}
          <div className="lg:col-span-7 bg-[#202024] rounded-[24px] p-6 sm:p-10 border border-white/5">
            <h3 className="text-xl font-bold text-white mb-2">
              Send an Inquiry
            </h3>
            <p className="text-xs text-zinc-400 mb-6 font-mono">
              Validated and processed by Go Backend API
            </p>

            {result && (
              <div className="mb-6 p-4 rounded-2xl bg-emerald-950/60 border border-emerald-800 text-emerald-300 text-sm flex items-start gap-3">
                <CheckCircle className="w-5 h-5 text-emerald-400 shrink-0 mt-0.5" />
                <div>
                  <div className="font-semibold text-white">Inquiry Received!</div>
                  <div className="text-xs text-emerald-200 mt-0.5">{result.message}</div>
                </div>
              </div>
            )}

            {errorMsg && (
              <div className="mb-6 p-4 rounded-2xl bg-rose-950/60 border border-rose-800 text-rose-300 text-sm flex items-center gap-3">
                <AlertCircle className="w-5 h-5 text-rose-400 shrink-0" />
                <span>{errorMsg}</span>
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-mono uppercase text-zinc-400 mb-2">
                    Name *
                  </label>
                  <input
                    type="text"
                    required
                    placeholder="e.g. Alex Morgan"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    className="w-full px-4 py-3 rounded-xl bg-zinc-900 border border-zinc-700 text-white placeholder-zinc-500 text-sm focus:outline-none focus:border-blue-500 transition-colors"
                  />
                </div>

                <div>
                  <label className="block text-xs font-mono uppercase text-zinc-400 mb-2">
                    Email *
                  </label>
                  <input
                    type="email"
                    required
                    placeholder="alex@company.com"
                    value={formData.email}
                    onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                    className="w-full px-4 py-3 rounded-xl bg-zinc-900 border border-zinc-700 text-white placeholder-zinc-500 text-sm focus:outline-none focus:border-blue-500 transition-colors"
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-mono uppercase text-zinc-400 mb-2">
                  Subject
                </label>
                <input
                  type="text"
                  placeholder="System Architecture / Collaboration"
                  value={formData.subject}
                  onChange={(e) => setFormData({ ...formData, subject: e.target.value })}
                  className="w-full px-4 py-3 rounded-xl bg-zinc-900 border border-zinc-700 text-white placeholder-zinc-500 text-sm focus:outline-none focus:border-blue-500 transition-colors"
                />
              </div>

              <div>
                <label className="block text-xs font-mono uppercase text-zinc-400 mb-2">
                  Message *
                </label>
                <textarea
                  rows={4}
                  required
                  placeholder="Tell me about your architecture goals or timeline..."
                  value={formData.message}
                  onChange={(e) => setFormData({ ...formData, message: e.target.value })}
                  className="w-full px-4 py-3 rounded-xl bg-zinc-900 border border-zinc-700 text-white placeholder-zinc-500 text-sm focus:outline-none focus:border-blue-500 transition-colors resize-none"
                />
              </div>

              <button
                type="submit"
                disabled={loading}
                className="w-full py-3.5 px-6 rounded-full bg-blue-600 hover:bg-blue-500 text-white font-semibold text-sm shadow-md shadow-blue-600/30 transition-all flex items-center justify-center gap-2 cursor-pointer disabled:opacity-60"
              >
                {loading ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" />
                    <span>Transmitting...</span>
                  </>
                ) : (
                  <>
                    <span>Submit Inquiry</span>
                    <Send className="w-4 h-4" />
                  </>
                )}
              </button>
            </form>
          </div>
        </div>
      </div>
    </section>
  );
};
