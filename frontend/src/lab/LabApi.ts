export const LAB_DISCLAIMER =
  'Independent redesign concept. Not affiliated with or endorsed by the original company.';

export interface LabProject {
  id: string;
  slug: string;
  title: string;
  original_product: string;
  disclaimer: string;
  focus?: string[];
  platforms?: string[];
  source_urls?: string[];
  status: string;
  featured: boolean;
  created_at: string;
  updated_at: string;
  cover_image?: string;
  summary?: string;
  case_study_url?: string;
  live_demo_url?: string;
}

export class LabApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = 'LabApiError';
    this.status = status;
  }
}

export function isPublishedProject(project: LabProject): boolean {
  return project.status.toUpperCase() === 'PUBLISHED';
}

export function onlyPublishedProjects(projects: LabProject[]): LabProject[] {
  return projects.filter(isPublishedProject);
}

async function readProjectsResponse(res: Response): Promise<LabProject[]> {
  if (!res.ok) {
    throw new LabApiError('Could not load the redesign catalog.', res.status);
  }
  const data: unknown = await res.json();
  if (!Array.isArray(data)) {
    throw new LabApiError('Could not load the redesign catalog.', res.status);
  }
  return (data as LabProject[]).filter(isPublishedProject);
}

export async function fetchLabCatalog(): Promise<LabProject[]> {
  let res: Response;
  try {
    res = await fetch('/api/lab/projects');
  } catch {
    throw new LabApiError('Could not load the redesign catalog.', 0);
  }
  return readProjectsResponse(res);
}

export async function fetchDesignLab(): Promise<LabProject[]> {
  let res: Response;
  try {
    res = await fetch('/api/portfolio/design-lab');
  } catch {
    throw new LabApiError('Could not load the redesign catalog.', 0);
  }
  return readProjectsResponse(res);
}

export async function fetchLabProject(slug: string): Promise<LabProject> {
  let res: Response;
  try {
    res = await fetch(`/api/lab/projects/${encodeURIComponent(slug)}`);
  } catch {
    throw new LabApiError('Could not load this case study.', 0);
  }
  if (res.status === 404) {
    throw new LabApiError('Project not found.', 404);
  }
  if (!res.ok) {
    throw new LabApiError('Could not load this case study.', res.status);
  }
  const project = (await res.json()) as LabProject;
  if (!isPublishedProject(project)) {
    throw new LabApiError('Project not found.', 404);
  }
  return project;
}
