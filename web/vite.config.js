import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
    plugins: [vue()],
    resolve: {
        alias: { '@': resolve(__dirname, 'src') }
    },
    build: {
        outDir: 'dist',
        emptyOutDir: true,
        sourcemap: false,
        rollupOptions: {
            output: {
                manualChunks: {
                    vue: ['vue', 'pinia'],
                    livekit: ['livekit-client']
                }
            }
        }
    },
    server: {
        port: 3000,
        proxy: {
            '/api': { target: 'http://localhost:8080', changeOrigin: true },
        }
    }
})
