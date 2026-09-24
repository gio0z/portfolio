// @ts-check
import { defineConfig } from 'astro/config'
import react from '@astrojs/react'
import sitemap from '@astrojs/sitemap'
import tailwindcss from '@tailwindcss/vite'

// Canonical origin for <link rel="canonical"> and sitemap entries. It must
// match PORTFOLIO_PUBLIC_ORIGIN, which the Go API uses for the same identity.
//
// This fails closed rather than defaulting to a placeholder host. A wrong
// canonical is worse than a build error: it tells search engines that the real
// pages live somewhere else, and nothing downstream would notice. Local builds
// that do not care about the origin can set PORTFOLIO_PUBLIC_ORIGIN=http://localhost:4321.
const rawSite = process.env.PORTFOLIO_PUBLIC_ORIGIN
if (!rawSite) {
  throw new Error(
    'PORTFOLIO_PUBLIC_ORIGIN is required to build: it becomes the canonical URL ' +
      'and sitemap origin. Example: PORTFOLIO_PUBLIC_ORIGIN=https://your-domain.example',
  )
}
const site = rawSite.replace(/\/+$/, '')

export default defineConfig({
  site,
  // The static build is served by the Go binary from a single directory, so the
  // build directory stays `dist` and the server needs no change to find it.
  outDir: './dist',
  build: {
    // Emit /about/index.html rather than /about.html, matching the directory
    // index resolution the Go static handler already performs.
    format: 'directory',
  },
  integrations: [
    react(),
    sitemap({
      // The admin area is an authenticated SPA with no indexable content.
      filter: (page) => !page.includes('/admin'),
    }),
  ],
  vite: {
    plugins: [tailwindcss()],
  },
})
