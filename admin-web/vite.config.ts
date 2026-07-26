import { defineConfig } from 'vite'
import { fileURLToPath, URL } from 'node:url'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    rolldownOptions: {
      output: {
        // 把体积大且变动少的公共库拆成独立 vendor chunk：主包更小、可长期缓存（配合路由懒加载）。
        manualChunks(id: string) {
          if (!id.includes('node_modules')) return
          if (id.includes('@tiptap') || id.includes('prosemirror')) return 'vendor-editor'
          if (id.includes('lucide')) return 'vendor-icons'
          if (id.includes('qrcode') || id.includes('echarts')) return 'vendor-viz'
          if (id.includes('vue') || id.includes('pinia') || id.includes('@vue')) return 'vendor-vue'
          return 'vendor'
        },
      },
    },
  },
  server: {
    proxy: {
      // 开发期把 /api 转发到 Go 后端，避免跨域
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      // 上传的图片走后端静态服务（/uploads/...），dev 期一并代理
      '/uploads': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
})
