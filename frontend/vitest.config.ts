/// <reference types="vitest/config" />
import { getViteConfig } from 'astro/config'

// getViteConfig inherits the Astro config, so tests resolve .astro imports and
// the same plugins the build uses. The jsdom environment and setup file keep
// the existing React component tests running unchanged.
export default getViteConfig({
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
  },
})
