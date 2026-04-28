import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  build: {
    // Output to the Go server's static directory for local development.
    // The Dockerfile overrides this path via sed for the Docker build context.
    outDir: '../api/static',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      // Proxy API and health requests to the Go server during local development
      '/api': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
    },
  },
})
