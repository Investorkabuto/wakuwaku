import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		// ブラウザだけで動くSPAとして書き出し、Firebase Hosting で配信する。
		adapter: adapter({ fallback: 'index.html' })
	}
};

export default config;
