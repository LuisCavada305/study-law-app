/// <reference types="vitest/config" />
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// Em desenvolvimento, /api vai para o servidor Go (make dev-api).
// API_URL troca o destino, se ele estiver em outra porta.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': process.env.API_URL ?? 'http://127.0.0.1:8080',
    },
  },
  test: {
    environment: 'node',
  },
})
