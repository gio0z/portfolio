import { Suspense, lazy } from 'react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { AdminLayout } from '../admin/AdminLayout'
import { AdminOverview } from '../admin/AdminOverview'
import { ReviewQueue } from '../admin/ReviewQueue'
import { ReviewDetail } from '../admin/ReviewDetail'

/**
 * The router for the two authenticated or data-driven areas of the site.
 *
 * Public pages are prerendered HTML and no longer route on the client, so they
 * are deliberately absent here: this router owns only the admin application and
 * the Design Lab.
 *
 * No basename is set because the existing route definitions and every Link in
 * the admin and lab components already use full absolute paths (/admin/reviews,
 * /lab/:slug). A basename would prefix them a second time.
 *
 * The Go server delivers the shell document for every /admin/* and /lab/* deep
 * link, so each URL is served the same bundle and routed here.
 */
const LazyLabCatalog = lazy(() => import('../lab/LabCatalog'))
const LazyLabCaseStudy = lazy(() => import('../lab/LabCaseStudy'))

export function LabShellFallback() {
  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold">Design Lab</h1>
      <p className="text-zinc-600">Loading the redesign catalog…</p>
    </div>
  )
}

export function IslandRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/admin" element={<AdminLayout />}>
          <Route index element={<AdminOverview />} />
          <Route path="overview" element={<AdminOverview />} />
          <Route path="reviews" element={<ReviewQueue />} />
          <Route path="reviews/:id" element={<ReviewDetail />} />
          <Route path="*" element={<AdminOverview />} />
        </Route>
        <Route
          path="/lab"
          element={
            <Suspense fallback={<LabShellFallback />}>
              <LazyLabCatalog />
            </Suspense>
          }
        />
        <Route
          path="/lab/:slug"
          element={
            <Suspense fallback={<LabShellFallback />}>
              <LazyLabCaseStudy />
            </Suspense>
          }
        />
      </Routes>
    </BrowserRouter>
  )
}

export default IslandRouter
