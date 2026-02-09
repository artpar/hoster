import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  // Served at root - Hoster serves its own embedded UI
  base: '/',
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      // Proxy through APIGate (front-facing server)
      // APIGate forwards to Hoster as upstream
      '/api': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
      // Auth endpoints served by APIGate module handler
      '/mod/auth': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
});
