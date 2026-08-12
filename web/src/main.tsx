// The latin-* entrypoints, not the bare 400.css. Those pull latin-ext,
// cyrillic and vietnamese for a console that ships no translations. Two
// weights is the whole set: 400 for everything, 600 for labels and <strong>.
// Importing here rather than in the theme keeps it in the entry chunk, so the
// stylesheet link lands in <head> and the preload scanner finds the font at
// the same time as the JS.
import '@fontsource/ibm-plex-mono/latin-400.css'
import '@fontsource/ibm-plex-mono/latin-600.css'

import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider, createRouter } from '@tanstack/react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ThemeProvider } from '@mui/material/styles'
import CssBaseline from '@mui/material/CssBaseline'
import { routeTree } from './routeTree.gen'
import { theme } from './theme'
import { ApiError } from '@/lib/http/errors'
import { RouteErrorFallback, RouteNotFound } from '@/components/feedback/States'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 10_000,
      // Retrying a 429 just extends the block, and a 404 will not fix itself.
      retry: (failureCount, error) => {
        if (error instanceof ApiError && error.status >= 400 && error.status < 500) return false
        return failureCount < 2
      },
      refetchOnWindowFocus: false,
    },
  },
})

const router = createRouter({
  routeTree,
  defaultPreload: 'intent',
  // On GitHub Pages the app is served from a repository subpath, so the router
  // has to strip that prefix before matching. Set by VITE_BASE in
  // web/.env.pages; BASE_URL is '/' everywhere else.
  basepath: import.meta.env.BASE_URL,
  // Applied to every route that does not define its own. Without them a render
  // throw shows a blank page and an unknown path shows an empty outlet.
  defaultErrorComponent: RouteErrorFallback,
  defaultNotFoundComponent: RouteNotFound,
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    {/*
      Dark by default rather than "system". ESTRAL is a dark palette first, so
      following the OS would drop anyone with a light desktop into the
      secondary look on their first visit. An explicit choice still wins and
      still persists, under localStorage's `mui-mode`.
    */}
    <ThemeProvider theme={theme} defaultMode="dark">
      <CssBaseline enableColorScheme />
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </ThemeProvider>
  </StrictMode>,
)
