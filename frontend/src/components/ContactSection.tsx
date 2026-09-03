import React, { useState } from 'react';
import { Mail, Phone, MapPin, Send, MessageSquare, CheckCircle, AlertCircle, Loader2 } from 'lucide-react';
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
    <section id="contact" className="py-24 relative overflow-hidden bg-gradient-to-t from-[#04060d] to-transparent">
      {/* Glow highlight */}
      <div className="absolute bottom-0 left-1/2 -translate-x-1/2 w-[600px] h-[300px] bg-blue-600/15 blur-[150px] rounded-full pointer-events-none" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        <div className="text-center max-w-3xl mx-auto mb-16">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-blue-950/80 border border-blue-800/40 text-blue-400 text-xs font-mono uppercase tracking-wider mb-4">
            <MessageSquare className="w-3.5 h-3.5" />
            <span>Initiate Collaboration</span>
          </div>
          <h2 className="text-3xl sm:text-5xl font-extrabold text-white tracking-tight">
            Let's Build Something High Impact
          </h2>
          <p className="text-slate-400 text-sm sm:text-base mt-3">
            Whether you need a rock-solid Go backend, modern reactive web architecture, or autonomous AI agent mesh, reach out directly.
          </p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-12 gap-10 max-w-6xl mx-auto">
          {/* Contact Details Cards */}
          <div className="lg:col-span-5 space-y-4">
            <div className="p-6 rounded-2xl bg-[#091124]/90 border border-blue-900/30 backdrop-blur-md">
              <h3 className="text-lg font-bold text-white mb-4 tracking-tight flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
                Direct Channels
              </h3>

              <div className="space-y-4">
                <a
                  href="mailto:regio@zoo.com"
                  className="flex items-center gap-4 p-3.5 rounded-xl bg-slate-900/60 hover:bg-blue-950/50 border border-slate-800/80 hover:border-blue-800/60 transition-all group"
                >
                  <div className="w-10 h-10 rounded-lg bg-blue-950 flex items-center justify-center text-blue-400 group-hover:text-blue-300">
                    <Mail className="w-5 h-5" />
                  </div>
                  <div className="truncate">
                    <div className="text-xs text-slate-400 font-mono">EMAIL</div>
                    <div className="text-sm font-semibold text-white group-hover:text-blue-300 truncate">regio@zoo.com</div>
                  </div>
                </a>

                <a
                  href="https://wa.me/6285156439303"
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center gap-4 p-3.5 rounded-xl bg-slate-900/60 hover:bg-blue-950/50 border border-slate-800/80 hover:border-blue-800/60 transition-all group"
                >
                  <div className="w-10 h-10 rounded-lg bg-emerald-950/70 border border-emerald-800/50 flex items-center justify-center text-emerald-400 group-hover:text-emerald-300">
                    <Phone className="w-5 h-5" />
                  </div>
                  <div>
                    <div className="text-xs text-slate-400 font-mono">WHATSAPP</div>
                    <div className="text-sm font-semibold text-white group-hover:text-emerald-300">+62 851-5643-9303</div>
                  </div>
                </a>

                <a
                  href="https://t.me/Ingouk_bot"
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center gap-4 p-3.5 rounded-xl bg-slate-900/60 hover:bg-blue-950/50 border border-slate-800/80 hover:border-blue-800/60 transition-all group"
                >
                  <div className="w-10 h-10 rounded-lg bg-sky-950/70 border border-sky-800/50 flex items-center justify-center text-sky-400 group-hover:text-sky-300">
                    <Send className="w-5 h-5" />
                  </div>
                  <div>
                    <div className="text-xs text-slate-400 font-mono">TELEGRAM BOT / CHANNEL</div>
                    <div className="text-sm font-semibold text-white group-hover:text-sky-300">@Ingouk_bot</div>
                  </div>
                </a>

                <div className="flex items-center gap-4 p-3.5 rounded-xl bg-slate-900/60 border border-slate-800/80">
                  <div className="w-10 h-10 rounded-lg bg-slate-950 flex items-center justify-center text-slate-400">
                    <MapPin className="w-5 h-5" />
                  </div>
                  <div>
                    <div className="text-xs text-slate-400 font-mono">LOCATION & ZONE</div>
                    <div className="text-sm font-semibold text-white">Indonesia (UTC+07:00 / WIB)</div>
                  </div>
                </div>
              </div>
            </div>

            <div className="p-5 rounded-2xl bg-gradient-to-br from-blue-950/50 to-[#091124] border border-blue-800/40 backdrop-blur-md">
              <div className="text-xs text-blue-300 font-mono uppercase tracking-wider mb-1">
                ⚡ Response SLA
              </div>
              <p className="text-xs text-slate-300 leading-relaxed">
                Direct inquiries are typically acknowledged within 2–4 hours. Priority routing active for production-critical inquiries.
              </p>
            </div>
          </div>

          {/* Interactive Form */}
          <div className="lg:col-span-7">
            <div className="p-7 sm:p-8 rounded-2xl bg-[#091124]/90 border border-blue-900/30 backdrop-blur-md shadow-2xl">
              <h3 className="text-xl font-bold text-white mb-2 tracking-tight">
                Send a Message
              </h3>
              <p className="text-xs text-slate-400 mb-6">
                Connected directly to the Go backend API with real-time validation.
              </p>

              {result && (
                <div className="mb-6 p-4 rounded-xl bg-emerald-950/50 border border-emerald-800/60 text-emerald-300 text-sm flex items-start gap-3">
                  <CheckCircle className="w-5 h-5 text-emerald-400 shrink-0 mt-0.5" />
                  <div>
                    <div className="font-semibold text-white">Message Dispatched!</div>
                    <div className="text-xs text-emerald-200/90 mt-0.5">{result.message}</div>
                    <div className="text-[11px] font-mono text-emerald-400/80 mt-1">Ref ID: {result.id}</div>
                  </div>
                </div>
              )}

              {errorMsg && (
                <div className="mb-6 p-4 rounded-xl bg-rose-950/50 border border-rose-800/60 text-rose-300 text-sm flex items-center gap-3">
                  <AlertCircle className="w-5 h-5 text-rose-400 shrink-0" />
                  <span>{errorMsg}</span>
                </div>
              )}

              <form onSubmit={handleSubmit} className="space-y-4">
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-mono uppercase tracking-wider text-slate-300 mb-1.5">
                      Your Name *
                    </label>
                    <input
                      type="text"
                      required
                      placeholder="e.g. Alex Morgan"
                      value={formData.name}
                      onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                      className="w-full px-4 py-2.5 rounded-xl bg-slate-900/90 border border-blue-900/40 text-white placeholder-slate-500 text-sm focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-colors"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-mono uppercase tracking-wider text-slate-300 mb-1.5">
                      Email Address *
                    </label>
                    <input
                      type="email"
                      required
                      placeholder="alex@company.com"
                      value={formData.email}
                      onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                      className="w-full px-4 py-2.5 rounded-xl bg-slate-900/90 border border-blue-900/40 text-white placeholder-slate-500 text-sm focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-colors"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-mono uppercase tracking-wider text-slate-300 mb-1.5">
                    Subject
                  </label>
                  <input
                    type="text"
                    placeholder="Project Inquiry / Engineering Consultation"
                    value={formData.subject}
                    onChange={(e) => setFormData({ ...formData, subject: e.target.value })}
                    className="w-full px-4 py-2.5 rounded-xl bg-slate-900/90 border border-blue-900/40 text-white placeholder-slate-500 text-sm focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-colors"
                  />
                </div>

                <div>
                  <label className="block text-xs font-mono uppercase tracking-wider text-slate-300 mb-1.5">
                    Message Details *
                  </label>
                  <textarea
                    rows={4}
                    required
                    placeholder="Briefly describe your objectives, architecture requirements, or timeline..."
                    value={formData.message}
                    onChange={(e) => setFormData({ ...formData, message: e.target.value })}
                    className="w-full px-4 py-2.5 rounded-xl bg-slate-900/90 border border-blue-900/40 text-white placeholder-slate-500 text-sm focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-colors resize-none"
                  />
                </div>

                <button
                  type="submit"
                  disabled={loading}
                  className="w-full py-3.5 px-6 rounded-xl bg-gradient-to-r from-blue-600 via-blue-700 to-indigo-600 hover:from-blue-500 hover:to-blue-600 text-white font-semibold text-sm shadow-xl shadow-blue-600/30 hover:shadow-blue-600/50 hover:-translate-y-0.5 transition-all duration-200 flex items-center justify-center gap-2 cursor-pointer disabled:opacity-60"
                >
                  {loading ? (
                    <>
                      <Loader2 className="w-4 h-4 animate-spin" />
                      <span>Transmitting Payload...</span>
                    </>
                  ) : (
                    <>
                      <Send className="w-4 h-4" />
                      <span>Transmit Message</span>
                    </>
                  )}
                </button>
              </form>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};
