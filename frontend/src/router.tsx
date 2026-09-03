import React, { Suspense, useEffect, useState } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Navbar } from './components/Navbar';
import { Hero } from './components/Hero';
import { TrustBar } from './components/TrustBar';
import { PhilosophySection } from './components/PhilosophySection';
import { ExpertiseSection } from './components/ExpertiseSection';
import { CoverflowSection } from './components/CoverflowSection';
import { ContactSection } from './components/ContactSection';
import { Footer } from './components/Footer';
import { AdminLayout } from './admin/AdminLayout';
import { AdminOverview } from './admin/AdminOverview';
import { ReviewQueue } from './admin/ReviewQueue';
import type { Profile, Project } from './types';

export function PortfolioShell() {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [projects, setProjects] = useState<Project[]>([]);

  useEffect(() => {
    fetch('/api/profile')
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data) setProfile(data);
      })
      .catch(() => {});

    fetch('/api/projects')
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (Array.isArray(data)) setProjects(data);
      })
      .catch(() => {});
  }, []);

  return (
    <div className="min-h-screen bg-[#F8F9FA] text-[#18181B] flex flex-col selection:bg-blue-600 selection:text-white">
      <Navbar />
      <main className="flex-grow">
        <Hero profile={profile} />
        <TrustBar />
        <PhilosophySection />
        <ExpertiseSection />
        <CoverflowSection projects={projects} />
        <ContactSection />
      </main>
      <Footer />
    </div>
  );
}

function LabShellFallback() {
  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold">Design Lab</h1>
      <p className="text-zinc-600">Design lab catalog</p>
    </div>
  );
}

const LazyLabShell = React.lazy(
  () =>
    Promise.resolve({
      default: LabShellFallback,
    })
);

export function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<PortfolioShell />} />
        <Route path="/admin" element={<AdminLayout />}>
          <Route index element={<AdminOverview />} />
          <Route path="overview" element={<AdminOverview />} />
          <Route path="reviews" element={<ReviewQueue />} />
          <Route path="*" element={<AdminOverview />} />
        </Route>
        <Route
          path="/lab/*"
          element={
            <Suspense fallback={<LabShellFallback />}>
              <LazyLabShell />
            </Suspense>
          }
        />
      </Routes>
    </BrowserRouter>
  );
}

export default AppRouter;
