import type { APIRoute } from 'astro'

// Generated rather than static so the Sitemap line always names the canonical
// origin from astro.config, instead of a host that was correct once.
export const GET: APIRoute = ({ site }) => {
  const origin = (site ?? new URL('https://regiodanipangestu.com')).origin

  const body = `# Public content is open to crawlers. The admin area is an
# authenticated application with no indexable content, and /api is a JSON
# surface that would only ever produce noise in search results.
User-agent: *
Allow: /
Disallow: /admin
Disallow: /api/

Sitemap: ${origin}/sitemap-index.xml
`

  return new Response(body, {
    headers: { 'Content-Type': 'text/plain; charset=utf-8' },
  })
}
