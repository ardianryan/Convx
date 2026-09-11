import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [svelte()],
  server: {
    port: 5147, // Default frontend dev port as requested
    host: '0.0.0.0',
    cors: true,
    proxy: {
      '/api': {
        target: 'http://localhost:7554',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: '../backend/dist',
    emptyOutDir: true,
  },
});
