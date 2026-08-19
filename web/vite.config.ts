import { defineConfig, loadEnv, type Plugin } from 'vite'
import react from '@vitejs/plugin-react'
import { tanstackRouter } from '@tanstack/router-plugin/vite'
import fs from 'node:fs'
import path from 'node:path'

/**
 * Two builds from one codebase:
 *
 *   vite build                  → live console, output into internal/spa/dist
 *                                 for the Go binary to embed
 *   vite build --mode pages     → static replay console for GitHub Pages,
 *                                 output into dist-pages
 *
 * Settings come from .env.pages via loadEnv rather than inline shell variables,
 * so the same command works in PowerShell, bash, and CI without cross-env.
 */
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, import.meta.dirname, '')

  const replay = env.VITE_REPLAY === '1'
  const outDir = replay ? 'dist-pages' : '../internal/spa/dist'

  return {
    // '/Go-Ledger/' on Pages, '/' everywhere else. Case matters: Pages serves
    // the project path exactly as the repository is named. Asset URLs and the
    // fixture paths in src/replay/fixtures.ts both derive from this.
    base: env.VITE_BASE || '/',
    plugins: [
      // Must run before the react plugin so generated routes are transformed.
      tanstackRouter({ target: 'react', autoCodeSplitting: true }),
      react(),
      pagesFallback(replay, path.resolve(import.meta.dirname, outDir)),
    ],
    resolve: {
      alias: { '@': path.resolve(import.meta.dirname, './src') },
    },
    server: {
      port: 5173,
      proxy: {
        '/api': { target: env.VITE_PROXY_TARGET || 'http://localhost:8080', changeOrigin: true },
        '/ops': { target: env.VITE_PROXY_TARGET || 'http://localhost:8080', changeOrigin: true },
      },
    },
    build: {
      outDir,
      emptyOutDir: true,
      // Maps cost 7.2 MB, over four times the rest of the bundle combined.
      // They also end up inside the server binary, since
      // internal/spa/embed_on.go embeds dist with `all:` and that takes the
      // .js.map files too. Dev builds get their maps from esbuild's transform
      // pipeline regardless of this flag, so turning it off costs nothing day
      // to day. Set VITE_SOURCEMAP=1 to debug a production build.
      sourcemap: env.VITE_SOURCEMAP === '1',
    },
  }
})

/**
 * GitHub Pages has no history fallback, so a deep link like /go-ledger/demos
 * misses and Pages serves 404.html. Making that file a copy of index.html lets
 * the SPA boot and route on the real path. The only cost is an HTTP 404 status
 * on a page that renders correctly.
 */
function pagesFallback(replay: boolean, outDir: string): Plugin {
  return {
    name: 'pages-spa-fallback',
    apply: 'build',
    closeBundle() {
      if (!replay) return

      const index = path.join(outDir, 'index.html')
      if (fs.existsSync(index)) {
        fs.copyFileSync(index, path.join(outDir, '404.html'))
      }
    },
  }
}
