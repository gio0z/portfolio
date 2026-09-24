// @ts-check
import { defineConfig } from 'astro/config'
import react from '@astrojs/react'
import sitemap from '@astrojs/sitemap'
import tailwindcss from '@tailwindcss/vite'

// Canonical origin for <link rel="canonical"> and sitemap entries. It must
// match PORTFOLIO_PUBLIC_ORIGIN, which the Go API uses as the same identity;
// the fallback keeps local builds honest rather than silently emitting
// localhost URLs into the sitemap.
const site = process.env.PORTFOLIO_PUBLIC_ORIGIN || 'https://regiodanipangestu.com'

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
