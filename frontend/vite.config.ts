import { defineConfig } from 'vite'
import react, { reactCompilerPreset } from '@vitejs/plugin-react'
import babel from '@rolldown/plugin-babel'
import tailwindcss from '@tailwindcss/vite'
import { tanstackRouter } from '@tanstack/router-plugin/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    tanstackRouter({ autoCodeSplitting: true }),
    babel({ presets: [reactCompilerPreset()] }),
    react(),
    tailwindcss(),
  ],
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:8090',
    },
  },
})
