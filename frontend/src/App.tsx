import { useEffect, useState } from 'react';
import { Navbar } from './components/Navbar';
import { Hero } from './components/Hero';
import { ProjectsSection } from './components/ProjectsSection';
import { SkillsSection } from './components/SkillsSection';
import { PhilosophySection } from './components/PhilosophySection';
import { ContactSection } from './components/ContactSection';
import { Footer } from './components/Footer';
import type { Profile, Project, SkillCategory } from './types';

// Fallback initial data in case server is warming up
const initialProfile: Profile = {
  name: 'Regio Dani Pangestu',
  tagline: 'Architecting Resilient Backend Engines & Autonomous AI Systems',
  title: 'Senior Full-Stack Engineer & AI Systems Architect',
  bio: 'Specializing in high-concurrency Go microservices, reactive modern web architectures (Vite/React/TypeScript), and autonomous multi-agent systems with Hermes Agent & LLM orchestration.',
  location: 'Indonesia',
  status: 'Available for High-Impact Projects',
  email: 'regio@zoo.com',
  phone: '+62 851-5643-9303',
  avatar: 'https://images.unsplash.com/photo-1534528741775-53994a69daeb?auto=format&fit=crop&w=600&q=80',
  stats: [
    { label: 'Experience', value: '5+ Years', sub: 'Full-Stack & Systems' },
    { label: 'Shipped Projects', value: '45+', sub: 'Production Grade' },
    { label: 'Uptime & Quality', value: '99.98%', sub: 'Zero-Downtime Releases' },
    { label: 'Code Coverage', value: '94%+', sub: 'Disciplined TDD' },
  ],
  social_links: {
    github: 'https://github.com/gio0z',
    linkedin: 'https://linkedin.com/in/regiodani',
    telegram: 'https://t.me/Ingouk_bot',
    whatsapp: 'https://wa.me/6285156439303',
  },
  highlights: [
    'Engineered distributed AI agent pipelines capable of multi-channel relay & autonomous execution',
    'Core advocate of deep modular design, strict test-first development, and clean architecture',
    'Proven track record building enterprise government booking systems, UMKM POS, and real-time gateways',
  ],
};

const initialProjects: Project[] = [
  {
    id: 'jam-nguar',
    title: 'Jam Nguar: Government Room Booking System',
    tagline: '4-tier RBAC room reservation & management engine for Blitar Regency ASN/PNS.',
    description: 'High-integrity municipal reservation platform handling strict booking deadlines, VIP room gating, automatic conflicts resolution, and cryptographic QR code check-in.',
    category: 'Full-Stack',
    tags: ['Rust', 'Actix-Web', 'Next.js', 'PostgreSQL', 'Tailwind CSS'],
    featured: true,
    github_url: 'https://github.com/gio0z/jam-nguar',
    demo_url: 'https://jam-nguar.blitar.go.id',
    image: 'https://images.unsplash.com/photo-1497366216548-37526070297c?auto=format&fit=crop&w=1200&q=80',
    metrics: '15k+ Monthly Bookings, Zero Double-Booking Guarantee',
  },
  {
    id: 'pos-kala',
    title: 'POS Kala: Unified UMKM Retail Engine',
    tagline: 'Point of sale, multi-warehouse inventory, and financial ledger platform for Indonesian SMEs.',
    description: 'Full-featured retail operations system with offline-first transaction sync, barcode scanning, thermal receipt printing, and comprehensive margin analysis.',
    category: 'Full-Stack',
    tags: ['Go', 'Vite', 'React', 'SQLite / Postgres', 'Tailwind'],
    featured: true,
    github_url: 'https://github.com/gio0z/pos-kala',
    demo_url: 'https://poskala.id',
    image: 'https://images.unsplash.com/photo-1556742049-0a67e557224f?auto=format&fit=crop&w=1200&q=80',
    metrics: 'Processed >$500k in GMV with sub-50ms checkout latency',
  },
  {
    id: 'hermes-agent-systems',
    title: 'Hermes Multi-Platform Gateway & Agent Mesh',
    tagline: 'Enterprise autonomous AI agent orchestration with multi-channel routing.',
    description: 'Architected bidirectional communication pipelines across WhatsApp Baileys, Telegram, Discord, and Slack with localized sandboxing, rate limiting, and long-term memory integration.',
    category: 'AI & Agents',
    tags: ['Python', 'Go', 'Docker', 'Node.js', 'Hermes Agent', 'Hindsight'],
    featured: true,
    github_url: 'https://github.com/gio0z/hermes-cs',
    demo_url: 'https://hermes-agent.nousresearch.com',
    image: 'https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?auto=format&fit=crop&w=1200&q=80',
    metrics: 'Active 24/7 autonomous triage, 10,000+ daily events handled',
  },
  {
    id: 'thundercrawl',
    title: 'ThunderCrawl: Distributed Web Scraping & Indexer',
    tagline: 'Ultra-fast asynchronous crawler with headless stealth browser cluster.',
    description: 'High-throughput content extraction engine designed for AI data ingestion, bypassing bot detections, parsing structured Markdown, and indexing knowledge graphs.',
    category: 'Systems',
    tags: ['Go', 'Chromium', 'Redis', 'ElasticSearch', 'Vite'],
    featured: false,
    github_url: 'https://github.com/gio0z/thundercrawl',
    demo_url: 'https://thundercrawl.dev',
    image: 'https://images.unsplash.com/photo-1558494949-ef010cbdcc31?auto=format&fit=crop&w=1200&q=80',
    metrics: '2,500 pages/minute crawling speed with <0.1% block rate',
  },
  {
    id: 'open-design-studio',
    title: 'Open Design Visual Artifact Engine',
    tagline: 'Live design-system generative renderer for agentic design systems.',
    description: 'Integrated with Open Design MCP daemon to build dynamic design systems, landing pages, and interactive prototypes directly from conversational agent instructions.',
    category: 'Frontend',
    tags: ['TypeScript', 'Vite', 'Tailwind CSS', 'MCP', 'Canvas'],
    featured: true,
    github_url: 'https://github.com/gio0z/open-design',
    demo_url: 'https://opendesign.studio',
    image: 'https://images.unsplash.com/photo-1507238691740-187a5b1d37b8?auto=format&fit=crop&w=1200&q=80',
    metrics: 'Sub-second live preview compilation across 50+ component variants',
  },
];

const initialSkills: SkillCategory[] = [
  {
    category: 'Backend & Systems',
    summary: 'High-performance concurrency, robust APIs, and clean domain design',
    skills: [
      { name: 'Go (Golang)', level: 95, proficiency: 'Expert', icon: 'Cpu', description: 'Goroutines, channels, microservices, net/http, standard library mastery' },
      { name: 'Rust', level: 85, proficiency: 'Advanced', icon: 'Shield', description: 'Memory safety, zero-cost abstractions, Actix-Web, CLI tooling' },
      { name: 'PostgreSQL / SQLite', level: 90, proficiency: 'Expert', icon: 'Database', description: 'Indexing, query optimization, migration management, schema design' },
      { name: 'RESTful & gRPC APIs', level: 94, proficiency: 'Expert', icon: 'Network', description: 'Strict contract design, idempotency, rate limiting, OpenAPI' },
    ],
  },
  {
    category: 'Frontend Engineering',
    summary: 'Fluid, responsive, accessible, and reactive user interfaces',
    skills: [
      { name: 'Vite Ecosystem', level: 95, proficiency: 'Expert', icon: 'Zap', description: 'HMR, optimized roll-up bundling, plugin architecture' },
      { name: 'React & TypeScript', level: 94, proficiency: 'Expert', icon: 'Code2', description: 'Custom hooks, state management, strict type checking, performance' },
      { name: 'Tailwind CSS', level: 96, proficiency: 'Expert', icon: 'Palette', description: 'Custom design systems, responsive grids, dark/blue theme styling' },
      { name: 'UI/UX & Pinterest Aesthetics', level: 90, proficiency: 'Advanced', icon: 'Layout', description: 'Modern glassmorphism, micro-interactions, clean typography' },
    ],
  },
  {
    category: 'AI & Autonomous Agents',
    summary: 'Agentic coding workflows, LLM orchestration, and multi-agent mesh',
    skills: [
      { name: 'Hermes Agent Framework', level: 95, proficiency: 'Expert', icon: 'Bot', description: 'Profiles, skills authoring, multi-platform gateway orchestration' },
      { name: 'Superpowers & Matt Pocock Flow', level: 92, proficiency: 'Expert', icon: 'Sparkles', description: 'Disciplined TDD, grilling, spec-driven development, deep modules' },
      { name: 'MCP (Model Context Protocol)', level: 90, proficiency: 'Expert', icon: 'Layers', description: 'Designing and integrating custom MCP tools and servers' },
      { name: 'Long-Term Memory Systems', level: 88, proficiency: 'Advanced', icon: 'Brain', description: 'Hindsight integration, semantic graphs, entity retrieval' },
    ],
  },
  {
    category: 'DevOps & Infrastructure',
    summary: 'Reliable continuous delivery, sandboxing, and Linux environments',
    skills: [
      { name: 'Docker & Containerization', level: 90, proficiency: 'Advanced', icon: 'Container', description: 'Multi-stage builds, rootless containers, compose clusters' },
      { name: 'Linux & WSL Administration', level: 92, proficiency: 'Expert', icon: 'Terminal', description: 'Shell automation, systemd services, process monitoring' },
      { name: 'CI/CD & Git Workflows', level: 92, proficiency: 'Expert', icon: 'GitBranch', description: 'GitHub Actions, automated test suites, release tagging' },
    ],
  },
];

export function App() {
  const [profile, setProfile] = useState<Profile>(initialProfile);
  const [projects, setProjects] = useState<Project[]>(initialProjects);
  const [skills, setSkills] = useState<SkillCategory[]>(initialSkills);

  useEffect(() => {
    // Fetch live profile from Go Backend API
    fetch('/api/profile')
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data && data.name) setProfile(data);
      })
      .catch(() => {
        // Fallback already initialized
      });

    // Fetch live projects from Go Backend API
    fetch('/api/projects')
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (Array.isArray(data) && data.length > 0) setProjects(data);
      })
      .catch(() => {});

    // Fetch live skills from Go Backend API
    fetch('/api/skills')
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (Array.isArray(data) && data.length > 0) setSkills(data);
      })
      .catch(() => {});
  }, []);

  return (
    <div className="min-h-screen bg-[#060913] text-slate-100 flex flex-col selection:bg-blue-600 selection:text-white">
      <Navbar status={profile.status} />
      <main className="flex-grow">
        <Hero profile={profile} />
        <ProjectsSection projects={projects} />
        <SkillsSection categories={skills} />
        <PhilosophySection />
        <ContactSection />
      </main>
      <Footer />
    </div>
  );
}

export default App;
