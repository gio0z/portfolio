import { describe, it, expect } from 'vitest'
import {
  SITE_NAME,
  SITE_TITLE,
  pageTitle,
  canonicalUrl,
  buildPageMeta,
  personJsonLd,
  creativeWorkJsonLd,
  toAbsolute,
} from './seo'
import type { Profile, Project } from '../types'

const ORIGIN = 'https://regiodanipangestu.com'

// Mirrors the live profile: no published email or phone.
const profile: Profile = {
  name: 'Regio Dani Pangestu',
  tagline: 'Software that keeps working after launch',
  bio: 'I design and build the systems small and mid-sized businesses run on.',
  title: 'Full-Stack Engineer',
  location: 'Indonesia',
  status: 'Available for new projects',
  email: '',
  phone: '',
  avatar: 'https://example.com/avatar.jpg',
  stats: [{ label: 'Production systems', value: '9', sub: 'Government, retail, travel' }],
  social_links: { github: 'https://github.com/gio0z' },
  highlights: ['Every release ships with a rollback plan'],
}

const project: Project = {
  id: 'jam-nguar',
  title: 'Jam Nguar',
  tagline: 'Room booking for Blitar Regency government staff.',
  description: 'Staff book meeting rooms against real availability.',
  category: 'Government',
  tags: ['Rust', 'Actix-Web'],
  featured: true,
  github_url: 'https://github.com/gio0z/jam-nguar',
  demo_url: 'https://github.com/gio0z/jam-nguar',
  image: 'https://example.com/jam.jpg',
  metrics: 'Handles booking conflicts',
}

describe('pageTitle', () => {
  it('appends the site name to a page name', () => {
    expect(pageTitle('About')).toBe(`About — ${SITE_NAME}`)
  })

  it('produces a distinct title for every public page', () => {
    const titles = ['About', 'Work', 'Services', 'Contact'].map(pageTitle)
    expect(new Set(titles).size).toBe(titles.length)
    for (const title of titles) {
      expect(title).not.toBe(SITE_TITLE)
      expect(title.length).toBeGreaterThan(SITE_NAME.length)
    }
  })

  it('never doubles the site name when a page passes it', () => {
    expect(pageTitle(SITE_NAME)).toBe(SITE_NAME)
  })
})

describe('canonicalUrl', () => {
  it('emits a trailing slash for directory routes', () => {
    expect(canonicalUrl(ORIGIN, '/about')).toBe(`${ORIGIN}/about/`)
    expect(canonicalUrl(ORIGIN, '/about/')).toBe(`${ORIGIN}/about/`)
    expect(canonicalUrl(ORIGIN, '/work/jam-nguar')).toBe(`${ORIGIN}/work/jam-nguar/`)
  })

  it('keeps the root bare', () => {
    expect(canonicalUrl(ORIGIN, '/')).toBe(`${ORIGIN}/`)
    expect(canonicalUrl(ORIGIN, '')).toBe(`${ORIGIN}/`)
  })

  it('leaves file paths alone', () => {
    expect(canonicalUrl(ORIGIN, '/sitemap.xml')).toBe(`${ORIGIN}/sitemap.xml`)
    expect(canonicalUrl(ORIGIN, '/robots.txt')).toBe(`${ORIGIN}/robots.txt`)
  })

  it('tolerates a trailing slash on the origin', () => {
    expect(canonicalUrl(`${ORIGIN}/`, '/about')).toBe(`${ORIGIN}/about/`)
  })

  it('tolerates a missing leading slash', () => {
    expect(canonicalUrl(ORIGIN, 'about')).toBe(`${ORIGIN}/about/`)
  })
})

describe('buildPageMeta', () => {
  it('produces a complete, self-consistent set of tags', () => {
    const meta = buildPageMeta({
      origin: ORIGIN,
      path: '/work/jam-nguar',
      title: pageTitle('Jam Nguar'),
      description: project.description,
    })

    expect(meta.title).toBe(`Jam Nguar — ${SITE_NAME}`)
    expect(meta.description).toBe(project.description)
    expect(meta.canonical).toBe(`${ORIGIN}/work/jam-nguar/`)
    expect(meta.ogType).toBe('website')
  })

  it('marks article pages as articles', () => {
    const meta = buildPageMeta({
      origin: ORIGIN,
      path: '/work/jam-nguar',
      title: pageTitle('Jam Nguar'),
      description: project.description,
      type: 'article',
    })
    expect(meta.ogType).toBe('article')
  })

  it('carries an absolute social image only when one is supplied', () => {
    const withImage = buildPageMeta({
      origin: ORIGIN,
      path: '/about',
      title: 'About',
      description: 'd',
      image: '/og/about.png',
    })
    expect(withImage.image).toBe(`${ORIGIN}/og/about.png`)

    const without = buildPageMeta({ origin: ORIGIN, path: '/about', title: 'About', description: 'd' })
    expect(without.image).toBeUndefined()
  })

  it('gives every route a distinct canonical URL', () => {
    const paths = ['/', '/about', '/work', '/work/jam-nguar', '/services', '/contact']
    const canonicals = paths.map(
      (path) => buildPageMeta({ origin: ORIGIN, path, title: `T${path}`, description: 'd' }).canonical,
    )
    expect(new Set(canonicals).size).toBe(paths.length)
  })
})

describe('structured data', () => {
  it('emits valid Person JSON-LD with the profile identity', () => {
    const parsed = JSON.parse(personJsonLd(profile, ORIGIN))
    expect(parsed['@context']).toBe('https://schema.org')
    expect(parsed['@type']).toBe('Person')
    expect(parsed.name).toBe(profile.name)
    expect(parsed.jobTitle).toBe(profile.title)
    expect(parsed.url).toBe(`${ORIGIN}/`)
    expect(parsed.sameAs).toContain('https://github.com/gio0z')
  })

  it('omits contact fields the profile does not publish', () => {
    const parsed = JSON.parse(personJsonLd(profile, ORIGIN))
    expect(parsed).not.toHaveProperty('telephone')
    expect(parsed).not.toHaveProperty('email')
  })

  it('emits valid CreativeWork JSON-LD per project', () => {
    const parsed = JSON.parse(creativeWorkJsonLd(project, ORIGIN))
    expect(parsed['@context']).toBe('https://schema.org')
    expect(parsed['@type']).toBe('CreativeWork')
    expect(parsed.name).toBe(project.title)
    expect(parsed.url).toBe(`${ORIGIN}/work/jam-nguar/`)
    expect(parsed.keywords).toEqual(project.tags)
    expect(parsed.author.name).toBe(SITE_NAME)
    expect(parsed.codeRepository).toBe(project.github_url)
  })
})

describe('toAbsolute', () => {
  it('leaves absolute URLs untouched', () => {
    expect(toAbsolute(ORIGIN, 'https://cdn.example.com/a.png')).toBe('https://cdn.example.com/a.png')
  })

  it('resolves site-relative paths against the origin', () => {
    expect(toAbsolute(ORIGIN, '/avatar.png')).toBe(`${ORIGIN}/avatar.png`)
  })
})
