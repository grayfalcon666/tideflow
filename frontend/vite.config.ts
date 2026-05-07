import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { quasar, transformAssetUrls } from '@quasar/vite-plugin'
import path from 'path'

export default defineConfig({
  // 强制开启开发模式
  mode: 'development',
  define: {
    __VUE_PROD_DEVTOOLS__: true,
    __VUE_OPTIONS_API__: true,
  },
  plugins: [
    vue({
      template: { transformAssetUrls }
    }),
    quasar({
      sassVariables: path.resolve(__dirname, 'src/styles/quasar-variables.sass')
    })
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src')
    }
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/videos': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/covers': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
