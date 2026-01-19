import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  // Production: served under /app/ via APIGate
  // Development: served at root with proxy
  base: process.env.NODE_ENV === 'production' ? '/app/' : '/',
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      // Proxy API calls to Hoster backend
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      // Proxy auth calls to APIGate (when running)
      '/auth': {
        target: 'http://localhost:8082',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/auth/, '/portal/api'),
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
});
