import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      // Forward every request that starts with /api to the gateway.
      '/api': 'http://localhost:8000',
    },
  },
})
