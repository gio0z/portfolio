# Frontend

The public site: an Astro 7 static build with React 19 islands, served by the Go
binary in the repository root. See the root `README.md` for the full picture.

```bash
npm ci

npm run dev      # dev server on http://localhost:4321 (proxies nothing: run the Go server for /api)
npm test         # vitest
npm run check    # astro check (types in .astro and .tsx)
npm run lint     # oxlint
npm run build    # content export + astro build + tsc
npm run preview  # serve the built output
```

## Pages

| Route | Rendered | JavaScript |
|---|---|---|
| `/` | prerendered | coverflow + expertise islands |
| `/about` | prerendered | none |
| `/work` | prerendered | none |
| `/work/<id>` | prerendered, one page per project | none |
| `/services` | prerendered | none |
| `/contact` | prerendered | the form |
| `/admin` | client-only shell (`noindex`) | the admin app |
| `/lab` | client-only shell (`noindex`) | the Design Lab |

Content pages are static HTML with their own metadata; they do not hydrate and
they do not fetch. `/admin` and `/lab` are single-page applications: React Router
owns every path beneath those prefixes, and the Go server returns the shell
document for each of them.

## Content

`src/content/portfolio.json` is **generated** from `pkg/api/content.go` and is not
hand-edited:

```bash
npm run prebuild   # go run ./tools/contentexport/main.go > src/content/portfolio.json
```

`npm run build` runs this first. `src/data/portfolio.ts` is the typed accessor
over that JSON.

## Conventions

- The build requires `PORTFOLIO_PUBLIC_ORIGIN`; it becomes the canonical URL and
  sitemap origin, so it fails closed rather than guessing a host. Development,
  tests, and type-checking do not need it.
- Zero warnings: `tsc`, `oxlint`, and `astro check` must all be clean. `.astro` is
  gitignored and excluded from linting because it is generated.
- Content pages must ship no JavaScript. The gate enforces this.
