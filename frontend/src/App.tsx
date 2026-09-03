import { useEffect, useState } from 'react';
import { Navbar } from './components/Navbar';
import { Hero } from './components/Hero';
import { TrustBar } from './components/TrustBar';
import { PhilosophySection } from './components/PhilosophySection';
import { ExpertiseSection } from './components/ExpertiseSection';
import { CoverflowSection } from './components/CoverflowSection';
import { ContactSection } from './components/ContactSection';
import { Footer } from './components/Footer';
import type { Profile, Project } from './types';

export function App() {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [projects, setProjects] = useState<Project[]>([]);

  useEffect(() => {
    // Fetch live profile from Go Backend API
    fetch('/api/profile')
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data) setProfile(data);
      })
      .catch(() => {});

    // Fetch live projects from Go Backend API
    fetch('/api/projects')
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (Array.isArray(data)) setProjects(data);
      })
      .catch(() => {});
  }, []);

  return (
    <div className="min-h-screen bg-[#F8F9FA] text-[#18181B] flex flex-col selection:bg-blue-600 selection:text-white">
      {/* 1. Header Navigation Bar */}
      <Navbar />

      <main className="flex-grow">
        {/* 2 & 3. Hero Section + Split Bento 50/50 Cards */}
        <Hero profile={profile} />

        {/* 4. Client / Tech Logos Strip */}
        <TrustBar />

        {/* 5. "My Philosophy" Section (3-Column Pillar Cards) */}
        <PhilosophySection />

        {/* 6. "Expertise" Section (Asymmetric Split + Horizontal Drawer Strip) */}
        <ExpertiseSection />

        {/* 7. 3D Stacked Coverflow / Layered Cards Showcase */}
        <CoverflowSection projects={projects} />

        {/* 8. Contact & Collaboration */}
        <ContactSection />
      </main>

      {/* 9. Minimal Editorial Footer */}
      <Footer />
    </div>
  );
}

export default App;
