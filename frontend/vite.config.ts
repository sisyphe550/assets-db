import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': path.resolve(__dirname, 'src') },
  },
  server: {
    port: 5173,
    proxy: {
      '/api/v1/user': 'http://localhost:8888',
      '/api/v1/asset': 'http://localhost:8889',
      '/api/v1/workflow': 'http://localhost:8890',
      '/api/v1/inventory': 'http://localhost:8891',
      '/api/v1/report': 'http://localhost:8892',
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom', 'react-router-dom'],
          antd: ['antd', '@ant-design/pro-components', '@ant-design/icons'],
        },
      },
    },
  },
});
