import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
	plugins: [react()],
	server: {
		allowedHosts: true,
		proxy: {
			'/api/v1': 'http://localhost:8080',
		},
	},
	test: {
		environment: 'jsdom',
		setupFiles: [],
	},
})
