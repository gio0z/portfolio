import { useEffect, useState } from 'react';
import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import { adminApi } from './AdminApi';
import type { AuthSession } from './types';

export function AdminLayout() {
  const [session, setSession] = useState<AuthSession | null>(null);
  const [loadingSession, setLoadingSession] = useState<boolean>(true);
  const navigate = useNavigate();

  useEffect(() => {
    let isMounted = true;
    adminApi
      .getSession()
      .then((sess) => {
        if (isMounted) {
          setSession(sess);
        }
      })
      .catch(() => {
        if (isMounted) {
          setSession({ authenticated: false });
        }
      })
      .finally(() => {
        if (isMounted) {
          setLoadingSession(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, []);

  const handleLogout = async () => {
    try {
      await adminApi.logout();
      setSession({ authenticated: false });
      navigate('/admin');
    } catch {
      // Ignore network errors on logout
      setSession({ authenticated: false });
    }
  };

  return (
    <div className="min-h-screen bg-zinc-50 text-zinc-900 flex flex-col selection:bg-blue-600 selection:text-white">
      {/* Admin Navbar */}
      <header className="bg-white border-b border-zinc-200 sticky top-0 z-30">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            {/* Branding & Main Title */}
            <div className="flex items-center gap-8">
              <div className="flex items-center gap-3">
                <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 ring-4 ring-emerald-100" />
                <h1 className="text-lg font-bold text-zinc-900 tracking-tight">
                  Admin Review Queue
                </h1>
              </div>

              {/* Navigation Tabs */}
              <nav className="hidden sm:flex items-center gap-1" aria-label="Admin Navigation">
                <NavLink
                  to="/admin"
                  end
                  className={({ isActive }) =>
                    `px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                      isActive
                        ? 'bg-zinc-100 text-zinc-900 font-semibold'
                        : 'text-zinc-600 hover:text-zinc-900 hover:bg-zinc-50'
                    }`
                  }
                >
                  Overview
                </NavLink>
                <NavLink
                  to="/admin/reviews"
                  className={({ isActive }) =>
                    `px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                      isActive
                        ? 'bg-zinc-100 text-zinc-900 font-semibold'
                        : 'text-zinc-600 hover:text-zinc-900 hover:bg-zinc-50'
                    }`
                  }
                >
                  Review Queue
                </NavLink>
              </nav>
            </div>

            {/* User Session & Actions */}
            <div className="flex items-center gap-4">
              <a
                href="/"
                className="text-xs text-zinc-500 hover:text-zinc-800 transition-colors hidden md:inline-block"
              >
                &larr; Back to Portfolio
              </a>

              {loadingSession ? (
                <div className="text-xs text-zinc-400">Loading session...</div>
              ) : session?.authenticated && session.login ? (
                <div className="flex items-center gap-3">
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-emerald-500" />
                    <span className="text-xs text-zinc-600">
                      Signed in as <strong className="text-zinc-900 font-semibold">{session.login}</strong>
                    </span>
                  </div>
                  <button
                    type="button"
                    onClick={handleLogout}
                    className="px-2.5 py-1 text-xs font-medium text-zinc-600 hover:text-red-600 hover:bg-red-50 rounded border border-zinc-200 transition-colors"
                  >
                    Sign out
                  </button>
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <span className="text-xs text-zinc-500">Not signed in</span>
                  <a
                    href="/api/admin/auth/login"
                    className="px-3 py-1 text-xs font-medium text-white bg-zinc-900 hover:bg-zinc-800 rounded-md transition-colors"
                  >
                    Sign in
                  </a>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Mobile Nav Tabs */}
        <div className="sm:hidden border-t border-zinc-100 px-4 py-2 flex items-center gap-2">
          <NavLink
            to="/admin"
            end
            className={({ isActive }) =>
              `px-3 py-1 text-xs font-medium rounded-md ${
                isActive ? 'bg-zinc-100 text-zinc-900 font-semibold' : 'text-zinc-600'
              }`
            }
          >
            Overview
          </NavLink>
          <NavLink
            to="/admin/reviews"
            className={({ isActive }) =>
              `px-3 py-1 text-xs font-medium rounded-md ${
                isActive ? 'bg-zinc-100 text-zinc-900 font-semibold' : 'text-zinc-600'
              }`
            }
          >
            Review Queue
          </NavLink>
        </div>
      </header>

      {/* Main Content Area */}
      <main className="max-w-7xl mx-auto w-full px-4 sm:px-6 lg:px-8 py-8 flex-1">
        <Outlet />
      </main>
    </div>
  );
}
