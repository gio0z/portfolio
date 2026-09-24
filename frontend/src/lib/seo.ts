import type { Profile, Project } from '../types'

/**
 * Metadata construction for the prerendered pages.
 *
 * Every value here ends up in the static HTML at build time, which is the whole
 * point of the migration: crawlers that never execute JavaScript must still see
 * a distinct title, description, and canonical URL per route.
 */

export const SITE_NAME = 'Regio Dani Pangestu'
export const SITE_TITLE = `${SITE_NAME} — Full-Stack Engineer`

/**
 * Builds a page title from the page's own name. The site name is appended once
 * and never twice, so a page may pass either its own name or the full title.
 */
export function pageTitle(page: string): string {
  if (page === SITE_NAME || page.startsWith(`${SITE_NAME} —`)) return page
  return `${page} — ${SITE_NAME}`
}

/**
 * Builds the canonical URL for a path.
 *
 * Astro emits directory output (`/about/index.html`), which resolves at
 * `/about/`. The canonical form carries that trailing slash so it matches the
 * URL the server actually serves, and file paths keep their extension.
 */
export function canonicalUrl(origin: string, path: string): string {
  const base = origin.replace(/\/+$/, '')
  let p = path.trim()
  if (p === '') return `${base}/`
  if (!p.startsWith('/')) p = `/${p}`
  if (p === '/') return `${base}/`
  if (/\.[a-z0-9]+$/i.test(p)) return `${base}${p}`
  return `${base}${p.replace(/\/+$/, '')}/`
}

/**
 * Resolves a URL against the canonical origin. Absolute URLs, including the
 * remote images the project records already point at, pass through unchanged.
 */
export function toAbsolute(origin: string, url: string): string {
  if (/^[a-z][a-z0-9+.-]*:\/\//i.test(url)) return url
  const base = origin.replace(/\/+$/, '')
  return `${base}${url.startsWith('/') ? url : `/${url}`}`
}

export interface PageMetaInput {
  origin: string
  path: string
  title: string
  description: string
  type?: 'website' | 'article'
  image?: string
}

export interface PageMeta {
  title: string
  description: string
  canonical: string
  ogType: string
  image?: string
}

export function buildPageMeta(input: PageMetaInput): PageMeta {
  return {
    title: input.title,
    description: input.description,
    canonical: canonicalUrl(input.origin, input.path),
    ogType: input.type ?? 'website',
    // Only emitted when a real image exists: a social card pointing at a
    // missing file renders worse than no card image at all.
    image: input.image ? toAbsolute(input.origin, input.image) : undefined,
  }
}

/** Person JSON-LD for the profile, skipping unpublished contact fields. */
export function personJsonLd(profile: Profile, origin: string): string {
  const data: Record<string, unknown> = {
    '@context': 'https://schema.org',
    '@type': 'Person',
    name: profile.name,
    jobTitle: profile.title,
    description: profile.bio,
    url: canonicalUrl(origin, '/'),
    address: { '@type': 'PostalAddress', addressCountry: profile.location },
    sameAs: Object.values(profile.social_links),
  }
  if (profile.avatar) data.image = toAbsolute(origin, profile.avatar)
  // The public profile deliberately publishes no email or phone; emitting an
  // empty value would advertise a contact route that does not exist.
  if (profile.email) data.email = profile.email
  if (profile.phone) data.telephone = profile.phone
  return JSON.stringify(data)
}

/** CreativeWork JSON-LD for one project case study. */
export function creativeWorkJsonLd(project: Project, origin: string): string {
  const data: Record<string, unknown> = {
    '@context': 'https://schema.org',
    '@type': 'CreativeWork',
    name: project.title,
    headline: project.tagline,
    description: project.description,
    url: canonicalUrl(origin, `/work/${project.id}`),
    keywords: project.tags,
    genre: project.category,
    author: { '@type': 'Person', name: SITE_NAME, url: canonicalUrl(origin, '/') },
  }
  if (project.image) data.image = toAbsolute(origin, project.image)
  if (project.github_url) data.codeRepository = project.github_url
  return JSON.stringify(data)
}
