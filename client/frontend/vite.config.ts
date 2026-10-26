import { defineConfig } from 'vite';

export default defineConfig({
  base: './',
  build: {
    outDir:    '../frontend/dist',
    emptyOutDir: true,
    sourcemap: false,
    rollupOptions: {
      output: {
        // Keep three-globe in its own chunk so the main bundle stays lean.
        manualChunks: (id) => {
          if (id.includes('three-globe') || id.includes('three/')) return 'three';
        },
      },
    },
  },
  clearScreen: false,
  server: {
    strictPort: true,
    port:       1420,
  },
  envPrefix: ['VITE_', 'TAURI_'],
});
