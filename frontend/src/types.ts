export interface StatMetric {
  label: string;
  value: string;
  sub: string;
}

export interface Profile {
  name: string;
  tagline: string;
  bio: string;
  title: string;
  location: string;
  status: string;
  email: string;
  /**
   * The public API marks the phone number `json:"-"` so it is never scrapeable,
   * which means the field is absent rather than empty in the payload.
   */
  phone?: string;
  avatar: string;
  stats: StatMetric[];
  social_links: Record<string, string>;
  highlights: string[];
}

export interface Project {
  id: string;
  title: string;
  tagline: string;
  description: string;
  category: string;
  tags: string[];
  featured: boolean;
  github_url: string;
  demo_url: string;
  image: string;
  metrics: string;
}

export interface DesignLabProject {
  id: string;
  slug: string;
  title: string;
  original_product: string;
  disclaimer: string;
  focus?: string[];
  platforms?: string[];
  status: string;
  featured: boolean;
  created_at?: string;
  updated_at?: string;
  cover_image?: string;
  summary?: string;
  case_study_url?: string;
  live_demo_url?: string;
}

export interface SkillItem {
  name: string;
  level: number;
  proficiency: string;
  icon: string;
  description: string;
}

export interface SkillCategory {
  category: string;
  summary: string;
  skills: SkillItem[];
}

export interface ContactFormData {
  name: string;
  email: string;
  subject: string;
  message: string;
}

export interface ContactResponse {
  success: boolean;
  message: string;
  id?: string;
  error?: string;
}
