// @ts-check
import { defineConfig } from 'astro/config'
import react from '@astrojs/react'
import sitemap from '@astrojs/sitemap'
import tailwindcss from '@tailwindcss/vite'

// Canonical origin for <link rel="canonical"> and sitemap entries. It must
// match PORTFOLIO_PUBLIC_ORIGIN, which the Go API uses for the same identity.
//
// A wrong canonical is worse than a missing build: it tells search engines the
// real pages live elsewhere, and nothing downstream would notice. So a real
// build requires it. Development, tests, and type-checking do not, and are left
// working without it — requiring the variable for `npm test` would make the
// value a nuisance to set rather than a thing anyone thinks about.
//
// The check runs at build time rather than at module scope because a failed
// import here would break every other command too.
const localDevOrigin = 'http://localhost:4321'

const localOriginRequired = {
  name: 'require-canonical-origin-for-build',
  hooks: {
    /** @param {{ command: 'dev' | 'build' | 'preview' | 'sync' }} options */
    'astro:config:setup': ({ command }) => {
      if (command === 'build' && !process.env.PORTFOLIO_PUBLIC_ORIGIN) {
        throw new Error(
          'PORTFOLIO_PUBLIC_ORIGIN is required to build: it becomes the canonical URL ' +
            'and sitemap origin. Example: PORTFOLIO_PUBLIC_ORIGIN=https://your-domain.example',
        )
      }
    },
  },
}

const site = (process.env.PORTFOLIO_PUBLIC_ORIGIN || localDevOrigin).replace(/\/+$/, '')

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
    localOriginRequired,
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
